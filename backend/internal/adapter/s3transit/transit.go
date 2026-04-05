package s3transit

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TransitStorage uploads and deletes objects in an AWS S3 bucket used for
// staging files to be analyzed by AWS services (e.g. Rekognition video).
type TransitStorage struct {
	client *s3.Client
	bucket string
}

// NewTransitStorage constructs an S3 transit adapter using the default AWS
// credential chain in the given region.
func NewTransitStorage(region, bucket string) (*TransitStorage, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("s3transit: load aws config: %w", err)
	}

	return &TransitStorage{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// Bucket returns the underlying S3 bucket name.
func (t *TransitStorage) Bucket() string {
	return t.bucket
}

// Upload streams a file into the transit bucket under the given key.
func (t *TransitStorage) Upload(
	ctx context.Context,
	key string,
	reader io.Reader,
	contentType string,
	size int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	input := &s3.PutObjectInput{
		Bucket:      aws.String(t.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	if _, err := t.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("s3transit put %q: %w", key, err)
	}
	return nil
}

// Delete removes an object from the transit bucket.
func (t *TransitStorage) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := t.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(t.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3transit delete %q: %w", key, err)
	}
	return nil
}
