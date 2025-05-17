package processor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rubichandrap/subvision/server/internal/minio"
)

var bucketName = os.Getenv("MINIO_BUCKET")
var tmpDir = os.Getenv("TMP_DIR")
var audioTmpDir = filepath.Join(tmpDir, "audio")

func ProcessUploadedFile(uploadID string, meta map[string]string) error {
	log.Printf("[Processor] Start processing uploadID %s with metadata %v\n", uploadID, meta)

	filename := meta["filename"]
	if filename == "" {
		return fmt.Errorf("missing filename in metadata")
	}

	// 1. Download video dari MinIO
	videoPath := filepath.Join(tmpDir, fmt.Sprintf("%s_%s", uploadID, filename))
	err := minio.DownloadFile(bucketName, uploadID, videoPath)
	if err != nil {
		return fmt.Errorf("failed to download file from MinIO: %w", err)
	}
	log.Printf("[Processor] Downloaded video to %s", videoPath)

	// 2. Convert video to .wav
	audioPath := filepath.Join(audioTmpDir, fmt.Sprintf("%s.wav", uploadID))
	err = convertToWav(videoPath, audioPath)
	if err != nil {
		return fmt.Errorf("failed to convert to wav: %w", err)
	}
	log.Printf("[Processor] Converted to WAV: %s", audioPath)

	// (Nanti: kirim ke whisper.cpp, generate .srt, gabung lagi pakai ffmpeg)

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
