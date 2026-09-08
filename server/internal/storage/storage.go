package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/rubichandrap/subvision/server/internal/handler"
)

// Client reads and deletes objects in the configured S3 bucket. Uploads
// arrive via tusd's s3store; the server reads them back and cleans them up
// when a Process is deleted.
type Client struct {
	s3     *s3.Client
	bucket string
}

func New(s3Client *s3.Client, bucket string) *Client {
	return &Client{s3: s3Client, bucket: bucket}
}

// Open streams the full object at key; the caller closes the reader.
func (c *Client) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	body, size, _, err := c.OpenRange(ctx, key, "")
	return body, size, err
}

// OpenRange streams a byte range of the object at key; the caller closes the reader.
// When byteRange is non-empty (e.g. "bytes=0-1024"), it is passed to S3's Range header,
// returning the body, the slice content length, and the Content-Range string (e.g. "bytes 0-1024/1048576").
// When byteRange is empty, it returns the full object with total size and an empty Content-Range string.
func (c *Client) OpenRange(ctx context.Context, key, byteRange string) (io.ReadCloser, int64, string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	if byteRange != "" {
		input.Range = aws.String(byteRange)
	}
	out, err := c.s3.GetObject(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "InvalidRange" || apiErr.ErrorCode() == "416") {
			return nil, 0, "", handler.ErrRangeUnsatisfiable
		}
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 416 {
			return nil, 0, "", handler.ErrRangeUnsatisfiable
		}
		return nil, 0, "", fmt.Errorf("failed to get object: %w", err)
	}
	return out.Body, aws.ToInt64(out.ContentLength), aws.ToString(out.ContentRange), nil
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

// Delete removes every object stored under the key prefix: the video itself
// and, for an Upload, the tusd s3store's `<id>.info` sibling that rides under
// the same prefix. S3 deletes are idempotent, so an already-gone object is
// not an error.
func (c *Client) Delete(ctx context.Context, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects under %s: %w", prefix, err)
		}
		if len(page.Contents) == 0 {
			continue
		}
		objects := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: obj.Key})
		}
		if _, err := c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(c.bucket),
			Delete: &types.Delete{Objects: objects},
		}); err != nil {
			return fmt.Errorf("failed to delete objects under %s: %w", prefix, err)
		}
	}
	return nil
}
