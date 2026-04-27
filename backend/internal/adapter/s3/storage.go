package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s *StorageService) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 download %q: %w", key, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 read body %q: %w", key, err)
	}
	return data, nil
}

type StorageService struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	publicURL string
}

func NewStorageService(
	endpoint string,
	accessKey string,
	secretKey string,
	bucket string,
	publicURL string,
	useSSL bool,
) *StorageService {
	scheme := "http"
	if useSSL {
		scheme = "https"
	}

	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(fmt.Sprintf("%s://%s", scheme, endpoint)),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		UsePathStyle: true, // Required for MinIO
	})

	presigner := s3.NewPresignClient(client)

	return &StorageService{
		client:    client,
		presigner: presigner,
		bucket:    bucket,
		publicURL: publicURL,
	}
}

func (s *StorageService) Upload(
	ctx context.Context,
	key string,
	reader io.Reader,
	contentType string,
	size int64,
) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload %q: %w", key, err)
	}

	return s.GetPublicURL(key), nil
}

func (s *StorageService) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete %q: %w", key, err)
	}
	return nil
}

func (s *StorageService) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/%s", s.publicURL, key)
}

func (s *StorageService) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	result, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign upload %q: %w", key, err)
	}

	return result.URL, nil
}

// GetPresignedDownloadURL returns a short-lived signed GET URL. Callers
// must perform the ownership check before issuing the URL — the URL
// itself encodes no authorization beyond the signed expiry.
func (s *StorageService) GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign download %q: %w", key, err)
	}
	return result.URL, nil
}

// GetPresignedDownloadURLAsAttachment is identical to
// GetPresignedDownloadURL but adds a `ResponseContentDisposition`
// override on the GetObject input. The S3-compatible signer adds it to
// the URL as the standard `response-content-disposition` query
// parameter, which R2/MinIO honor by replaying as the response header.
// The browser then saves the file under `filename` instead of
// rendering it inline. The filename is quoted to keep spaces/special
// characters safe; callers should pass a sanitized base filename
// (e.g. "FAC-000123.pdf").
func (s *StorageService) GetPresignedDownloadURLAsAttachment(
	ctx context.Context,
	key string,
	filename string,
	expiry time.Duration,
) (string, error) {
	disposition := fmt.Sprintf("attachment; filename=%q", filename)
	result, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(disposition),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign download as attachment %q: %w", key, err)
	}
	return result.URL, nil
}
