package main

import (
	"context"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/handler"
	"github.com/rubichandrap/subvision/server/internal/minio"
	"github.com/rubichandrap/subvision/server/internal/processor"
	"github.com/rubichandrap/subvision/server/internal/rabbitmq"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

func main() {
	env := config.LoadEnv()

	// Initialize MinIO client
	minio.Init(env.MinioEndpoint, env.MinioAccessKey, env.MinioSecretKey)

	// Init RabbitMQ connection, publisher, and consumer
	conn := rabbitmq.Connect(env.AmqpURL)
	defer conn.Close()

	publisher := rabbitmq.NewPublisher(conn)
	rabbitmq.StartConsumer(conn, func(payload rabbitmq.JobPayload) {
		processor.ProcessUploadedFile(payload.UploadID, payload.Meta)
	})

	// AWS-style config for MinIO
	awsCfg, err := awsCfg.LoadDefaultConfig(context.TODO(),
		awsCfg.WithRegion("us-east-1"),
		awsCfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(env.MinioAccessKey, env.MinioSecretKey, "")),
		awsCfg.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               "http://" + env.MinioEndpoint, // use https if MinIO is secured
				SigningRegion:     "us-east-1",                   // static region for signing
				HostnameImmutable: true,                          // required for MinIO
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("❌ Failed to load AWS config for MinIO: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	// Set up s3store
	store := s3store.New(env.MinioBucket, s3Client)

	// Compose the store and locker
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	// Create the tusd handler
	tusdHandler, err := tusd.NewHandler(tusd.Config{
		BasePath:              "/files/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})
	if err != nil {
		log.Fatalf("Unable to create tusd handler: %s", err)
	}

	// Listen for completed uploads
	go func() {
		for {
			event := <-tusdHandler.CompleteUploads
			log.Printf("Upload %s finished\n", event.Upload.ID)

			publisher.PublishJob(event.Upload.ID, event.Upload.MetaData)
		}
	}()

	// Set up Gin
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{env.ClientURL},
	}))

	// Register tusd handler
	handler.RegisterTusd(r, env.ClientURL, tusdHandler)

	log.Println("🚀 Starting Subvision backend on port", env.Port)
	if err := r.Run(":" + env.Port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
