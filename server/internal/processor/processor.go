package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/rabbitmq"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

var env = config.LoadEnv()

var videoTmpDir = filepath.Join(env.TmpDir, "videos")
var audioTmpDir = filepath.Join(env.TmpDir, "audios")

type ObjectStore interface {
	Upload(ctx context.Context, key, filePath string) error
	Download(ctx context.Context, key, destPath string) error
}

func ProcessUploadedFile(vfxPublisher *rabbitmq.GenerateVfxJobPublisher, store ObjectStore, payload rabbitmq.UploadJobPayload) error {
	ctx := context.Background()
	uploadID := payload.UploadID
	storage := payload.Storage
	meta := payload.Meta

	log.Printf("[Processor] Start processing uploadID %s, with st%v\n, with metadata %v\n", uploadID, storage, meta)

	key := storage["Key"]
	if key == "" {
		return fmt.Errorf("missing key in storage")
	}

	id := strings.Split(key, "/")[1]

	// Download video
	videoPath := filepath.Join(videoTmpDir, id)
	if err := store.Download(ctx, key, videoPath); err != nil {
		return fmt.Errorf("failed to download video from object storage: %w", err)
	}
	log.Printf("[Processor] Downloaded video to %s", videoPath)

	// Convert video to wav
	audioPath := filepath.Join(audioTmpDir, fmt.Sprintf("%s.wav", id))
	if err := convertToWav(videoPath, audioPath); err != nil {
		return fmt.Errorf("failed to convert to wav: %w", err)
	}
	log.Printf("[Processor] Converted to WAV: %s", audioPath)

	// Transcribe using whisper
	modelPath := env.WhisperModelPath
	segments, err := transcriber.Transcribe(modelPath, audioPath)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}

	vfxPublisher.Publish(rabbitmq.GenerateVfxJobPayload{
		ObjectKey: key,
		Segments:  segments,
	})

	return nil
}

func convertToWav(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[ffmpeg] Running conversion command: %v", cmd.Args)
	return cmd.Run()
}
