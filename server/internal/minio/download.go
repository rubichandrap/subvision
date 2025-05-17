package minio

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/minio/minio-go/v7"
)

func DownloadFile(bucket, objectName, destPath string) error {
	ctx := context.Background()
	object, err := Client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, object)
	if err != nil {
		return fmt.Errorf("failed to copy object to file: %w", err)
	}

	return nil
}
