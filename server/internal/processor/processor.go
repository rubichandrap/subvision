package processor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/minio"
	"github.com/rubichandrap/subvision/server/internal/rabbitmq"
	"github.com/rubichandrap/subvision/server/internal/subtitle"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

var env = config.LoadEnv()

var videoTmpDir = filepath.Join(env.TmpDir, "videos")
var audioTmpDir = filepath.Join(env.TmpDir, "audios")
var subtitleTmpDir = filepath.Join(env.TmpDir, "subtitles")

func ProcessUploadedFile(payload rabbitmq.UploadJobPayload) error {
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
	if err := minio.DownloadFile(env.MinioBucket, key, videoPath); err != nil {
		return fmt.Errorf("failed to download file from MinIO: %w", err)
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

	srtPath := filepath.Join(subtitleTmpDir, fmt.Sprintf("%s.srt", id))
	srtFile, err := os.Create(srtPath)
	if err != nil {
		return fmt.Errorf("failed to create srt file: %w", err)
	}
	defer srtFile.Close()

	subs := convertSegments(segments)
	if err := subtitle.SRTFromSegments(srtFile, subs); err != nil {
		return fmt.Errorf("failed to write srt file: %w", err)
	}

	log.Printf("[Processor] Subtitle saved to %s", srtPath)

	log.Printf("[Processor] Finished processing uploadID %s\n", uploadID)
	return nil
}

func convertToWav(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("[ffmpeg] Running conversion command: %v", cmd.Args)
	return cmd.Run()
}

func convertSegments(src []transcriber.Segment) []subtitle.Segment {
	dst := make([]subtitle.Segment, len(src))
	for i, s := range src {
		dst[i] = subtitle.Segment{
			Start: s.Start,
			End:   s.End,
			Text:  s.Text,
		}
	}
	return dst
}
