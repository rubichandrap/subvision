package storage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client retrieves objects from the configured S3 bucket. Uploads arrive via
// tusd's s3store; the server only reads them back.
type Client struct {
	s3     *s3.Client
	bucket string
}

func New(s3Client *s3.Client, bucket string) *Client {
	return &Client{s3: s3Client, bucket: bucket}
}

// Open streams the object at key; the caller closes the reader.
func (c *Client) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get object: %w", err)
	}
	return out.Body, aws.ToInt64(out.ContentLength), nil
}

func (c *Client) Download(ctx context.Context, key, destPath string) error {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object: %w", err)
	}
	defer out.Body.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, out.Body); err != nil {
		return fmt.Errorf("failed to copy object to file: %w", err)
	}

	return nil
}
