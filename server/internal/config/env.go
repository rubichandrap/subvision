package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

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
	VADGating        bool
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
		// Optional (ADR-0007): unset keeps gating off — the pre-gating behavior.
		VADGating: VADGatingFromEnv(),
	}
}

// VADGatingFromEnv parses the optional VAD_GATING flag shared by the server
// and the onset fixture command. A set-but-invalid value fails loudly:
// misconfiguration must never silently change what gets transcribed.
func VADGatingFromEnv() bool {
	if raw := os.Getenv("VAD_GATING"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			log.Fatalf("Invalid VAD_GATING value %q: use \"true\" or \"false\"", raw)
		}
		return parsed
	}
	return false
}
