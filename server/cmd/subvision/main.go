package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/db"
	"github.com/rubichandrap/subvision/server/internal/handler"
	"github.com/rubichandrap/subvision/server/internal/job"
	"github.com/rubichandrap/subvision/server/internal/processor"
	"github.com/rubichandrap/subvision/server/internal/storage"
	"github.com/rubichandrap/subvision/server/internal/rabbitmq"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
	"github.com/rubichandrap/subvision/server/internal/utils"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"

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

	utils.EnsureDirs(tmpDir, videoTmpDir, audioTmpDir, subtitleTmpDir, outputsTmpDir, "data")

	// S3 config (RustFS or any S3-compatible store)
	awsCfg, err := awsCfg.LoadDefaultConfig(context.TODO(),
		awsCfg.WithRegion("us-east-1"),
		awsCfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(env.S3AccessKey, env.S3SecretKey, "")),
		awsCfg.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               env.S3Endpoint, // use https if the store is secured
				SigningRegion:     "us-east-1",    // static region for signing
				HostnameImmutable: true,
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config for S3: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)
	objectStore := storage.New(s3Client, env.S3Bucket)

	// Job state: the Process lifecycle, persisted so restarts don't lose it.
	database, err := db.Open("data/subvision.db")
	if err != nil {
		log.Fatalf("Failed to open job database: %v", err)
	}
	defer database.Close()
	jobs, err := job.NewStore(database)
	if err != nil {
		log.Fatalf("Failed to prepare job store: %v", err)
	}

	// Init RabbitMQ connection, publisher, and consumer
	conn := rabbitmq.Connect(env.AmqpURL)
	defer conn.Close()

	// publishers
	uploadJobPublisher := rabbitmq.NewUploadJobPublisher(conn)
	vfxJobPublisher := rabbitmq.NewVfxJobPublisher(conn)

	// consumers
	proc := processor.New(processor.Options{
		Publisher:        vfxJobPublisher,
		Store:            objectStore,
		Transcribe:       transcriber.Transcribe,
		TmpDir:           env.TmpDir,
		WhisperModelPath: env.WhisperModelPath,
		Lifecycle:        jobs,
	})
	uploadJobConsumer := rabbitmq.NewUploadJobConsumer(conn)
	err = uploadJobConsumer.Start(func(payload rabbitmq.UploadJobPayload) {
		key := payload.Storage["Key"]
		if key == "" {
			log.Printf("[UploadJobConsumer] upload job %q carries no object key: %+v", payload.UploadID, payload)
			failJob(jobs, payload.UploadID, "upload job carried no object key")
			return
		}
		if err := proc.ProcessUploadedFile(payload.UploadID, key); err != nil {
			log.Printf("[Processor] Error: %v", err)
			// A transcription that fails must surface as a failed process,
			// not leave the job in-flight forever.
			failJob(jobs, payload.UploadID, err.Error())
		}
	})
	if err != nil {
		log.Fatalf("failed to consume upload job: %s", err)
	}

	completedConsumer := rabbitmq.NewJobCompletedConsumer(conn)
	err = completedConsumer.Start(func(event vfxjob.JobCompleted) (bool, error) {
		return jobs.MarkDone(event.UploadID, event.OutputKey)
	})
	if err != nil {
		log.Fatalf("failed to consume job_completed events: %s", err)
	}

	failedConsumer := rabbitmq.NewJobFailedConsumer(conn)
	err = failedConsumer.Start(func(event vfxjob.JobFailed) (bool, error) {
		return jobs.MarkFailed(event.UploadID, event.Reason)
	})
	if err != nil {
		log.Fatalf("failed to consume job_failed events: %s", err)
	}

	// Set up s3store
	s3Store := s3store.New(env.S3Bucket, s3Client)

	// Compose the store and locker
	composer := tusd.NewStoreComposer()
	s3Store.ObjectPrefix = config.ObjectPrefix
	s3Store.UseIn(composer)

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

			// Record the process before the job enters the queue so the
			// client can already find it when it starts tracking.
			if err := jobs.Create(event.Upload.ID, event.Upload.MetaData["filename"]); err != nil {
				log.Printf("failed to record job for upload %s: %v", event.Upload.ID, err)
			}

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
	handler.RegisterTusd(r, tusdHandler)

	// Register the read-only status API over the Process lifecycle
	handler.RegisterJobs(r, jobs, objectStore)

	log.Println("Starting Subvision backend on port", env.Port)
	if err := r.Run(":" + env.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// failJob records why an upload never made it past the server's own pipeline
// stages. A job the store doesn't know (or that is already terminal) is
// logged, not retried.
func failJob(jobs *job.Store, uploadID, reason string) {
	recorded, err := jobs.MarkFailed(uploadID, reason)
	if err != nil {
		log.Printf("[Job] Failed to mark job %s as failed: %v", uploadID, err)
	} else if !recorded {
		log.Printf("[Job] Job %s unknown or terminal, failure reason not recorded: %s", uploadID, reason)
	}
}
