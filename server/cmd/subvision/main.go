package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/handler"
	"github.com/rubichandrap/subvision/server/internal/minio"
	"github.com/rubichandrap/subvision/server/internal/processor"
	"github.com/rubichandrap/subvision/server/internal/rabbitmq"
	"github.com/rubichandrap/subvision/server/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

func main() {
	env := config.LoadEnv()

	tmpDir := env.TmpDir
	videoTmpDir := filepath.Join(tmpDir, "videos")
	audioTmpDir := filepath.Join(tmpDir, "audios")
	subtitleTmpDir := filepath.Join(tmpDir, "subtitles")
	outputsTmpDir := filepath.Join(tmpDir, "outputs")

	utils.EnsureDirs(tmpDir, videoTmpDir, audioTmpDir, subtitleTmpDir, outputsTmpDir)

	// Initialize MinIO client
	minio.Init(env.MinioEndpoint, env.MinioAccessKey, env.MinioSecretKey)

	// Init RabbitMQ connection, publisher, and consumer
	conn := rabbitmq.Connect(env.AmqpURL)
	defer conn.Close()

	uploadJobPublisher := rabbitmq.NewUploadJobPublisher(conn)
	uploadJobConsumer := rabbitmq.NewUploadJobConsumer(conn)
	err := uploadJobConsumer.Start(func(payload rabbitmq.UploadJobPayload) {
		if err := processor.ProcessUploadedFile(rabbitmq.UploadJobPayload{
			UploadID: payload.UploadID,
			Storage:  payload.Storage,
			Meta:     payload.Meta,
		}); err != nil {
			log.Printf("[Processor] Error: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("failed to consume upload job: %s", err)
	}

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
		log.Fatalf("Failed to load AWS config for MinIO: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	// Set up s3store
	store := s3store.New(env.MinioBucket, s3Client)

	// Compose the store and locker
	composer := tusd.NewStoreComposer()
	store.ObjectPrefix = config.ObjectPrefix
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

			log.Printf("[Debug] Expected MinIO key: %s%s", config.ObjectPrefix, event.Upload.Storage["Key"])

			err := uploadJobPublisher.Publish(rabbitmq.UploadJobPayload{
				UploadID: event.Upload.ID,
				Meta:     event.Upload.MetaData,
				Storage:  event.Upload.Storage,
			})
			if err != nil {
				log.Printf("failed to publish upload job: %v", err)
			}
		}
	}()

	// Set up Gin
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{env.ClientURL},
		AllowMethods:     []string{"POST", "GET", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization", "Upload-Length", "Tus-Resumable", "Upload-Metadata", "Upload-Offset"},
		ExposeHeaders:    []string{"Location", "Upload-Offset", "Upload-Length", "Tus-Resumable"},
		AllowCredentials: true,
	}))
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Register tusd handler
	handler.RegisterTusd(r, env.ClientURL, tusdHandler)

	log.Println("Starting Subvision backend on port", env.Port)
	if err := r.Run(":" + env.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
