package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	Port             string
	TmpDir           string
	ClientURL        string
	AmqpURL          string
	MinioEndpoint    string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioBucket      string
	WhisperModelPath string
}

func LoadEnv() *Env {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	requiredVars := []string{
		"PORT", "TMP_DIR",
		"CLIENT_URL",
		"MINIO_HOST", "MINIO_PORT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_BUCKET",
		"RABBITMQ_HOST", "RABBITMQ_PORT", "RABBITMQ_USER", "RABBITMQ_PASSWORD",
		"WHISPER_MODEL_PATH",
	}
	for _, key := range requiredVars {
		if os.Getenv(key) == "" {
			log.Fatalf("Missing required environment variable: %s", key)
		}
	}

	rabbitUser := os.Getenv("RABBITMQ_USER")
	rabbitPass := os.Getenv("RABBITMQ_PASSWORD")
	rabbitHost := os.Getenv("RABBITMQ_HOST")
	rabbitPort := os.Getenv("RABBITMQ_PORT")
	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitUser, rabbitPass, rabbitHost, rabbitPort)

	minioHost := os.Getenv("MINIO_HOST")
	minioPort := os.Getenv("MINIO_PORT")
	minioEndpoint := fmt.Sprintf("%s:%s", minioHost, minioPort)

	return &Env{
		Port:             os.Getenv("PORT"),
		TmpDir:           os.Getenv("TMP_DIR"),
		ClientURL:        os.Getenv("CLIENT_URL"),
		AmqpURL:          amqpURL,
		MinioEndpoint:    minioEndpoint,
		MinioAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:      os.Getenv("MINIO_BUCKET"),
		WhisperModelPath: os.Getenv("WHISPER_MODEL_PATH"),
	}
}
