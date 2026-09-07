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
	S3Endpoint       string
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string
	WhisperModelPath string
}

func LoadEnv() *Env {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	requiredVars := []string{
		"PORT", "TMP_DIR",
		"CLIENT_URL",
		"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET",
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

	return &Env{
		Port:             os.Getenv("PORT"),
		TmpDir:           os.Getenv("TMP_DIR"),
		ClientURL:        os.Getenv("CLIENT_URL"),
		AmqpURL:          amqpURL,
		S3Endpoint:       os.Getenv("S3_ENDPOINT"),
		S3AccessKey:      os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:      os.Getenv("S3_SECRET_KEY"),
		S3Bucket:         os.Getenv("S3_BUCKET"),
		WhisperModelPath: os.Getenv("WHISPER_MODEL_PATH"),
	}
}
