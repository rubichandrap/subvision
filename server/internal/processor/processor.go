package processor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rubichandrap/subvision/server/internal/config"
	"github.com/rubichandrap/subvision/server/internal/minio"
	"github.com/rubichandrap/subvision/server/internal/subtitle"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

var bucketName = os.Getenv("MINIO_BUCKET")
var tmpDir = os.Getenv("TMP_DIR")
var videoTmpDir = filepath.Join(tmpDir, "videos")
var audioTmpDir = filepath.Join(tmpDir, "audios")
var subtitleTmpDir = filepath.Join(tmpDir, "subtitles")

func ProcessUploadedFile(uploadID string, meta map[string]string) error {
	log.Printf("[Processor] Start processing uploadID %s with metadata %v\n", uploadID, meta)

	filename := meta["filename"]
	if filename == "" {
		return fmt.Errorf("missing filename in metadata")
	}

	// Download video
	videoPath := filepath.Join(videoTmpDir, fmt.Sprintf("%s_%s", uploadID, filename))
	if err := minio.DownloadFile(bucketName, config.ObjectPrefix+uploadID, videoPath); err != nil {
		return fmt.Errorf("failed to download file from MinIO: %w", err)
	}
	log.Printf("[Processor] Downloaded video to %s", videoPath)

	// Convert video to wav
	audioPath := filepath.Join(audioTmpDir, fmt.Sprintf("%s.wav", uploadID))
	if err := convertToWav(videoPath, audioPath); err != nil {
		return fmt.Errorf("failed to convert to wav: %w", err)
	}
	log.Printf("[Processor] Converted to WAV: %s", audioPath)

	// Transcribe using whisper
	modelPath := os.Getenv("WHISPER_MODEL_PATH")
	segments, err := transcriber.Transcribe(modelPath, audioPath)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}

	srtPath := filepath.Join(subtitleTmpDir, fmt.Sprintf("%s.srt", uploadID))
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
