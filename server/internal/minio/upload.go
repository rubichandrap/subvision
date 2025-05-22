package minio

import (
	"context"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
)

func UploadFile(bucketName string, objectName string, filePath string) error {
	ctx := context.Background()
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	_, err = Client.PutObject(ctx, bucketName, objectName, file, info.Size(), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}
