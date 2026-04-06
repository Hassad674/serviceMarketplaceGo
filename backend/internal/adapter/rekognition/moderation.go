package rekognition

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	rtypes "github.com/aws/aws-sdk-go-v2/service/rekognition/types"

	"marketplace-backend/internal/domain/media"
	portservice "marketplace-backend/internal/port/service"
)

// ModerationServiceDeps groups the configuration required to construct a ModerationService.
type ModerationServiceDeps struct {
	Region      string
	Threshold   float64
	SNSTopicARN string
	RoleARN     string
}

// ModerationService implements ContentModerationService using AWS Rekognition.
type ModerationService struct {
	client      *rekognition.Client
	threshold   float64
	snsTopicARN string
	roleARN     string
}

// NewModerationService creates a Rekognition-backed moderation service.
// snsTopicARN and roleARN are only required for async video moderation.
func NewModerationService(deps ModerationServiceDeps) (*ModerationService, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(deps.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("rekognition: load aws config: %w", err)
	}

	client := rekognition.NewFromConfig(cfg)
	return &ModerationService{
		client:      client,
		threshold:   deps.Threshold,
		snsTopicARN: deps.SNSTopicARN,
		roleARN:     deps.RoleARN,
	}, nil
}

// AnalyzeImage sends image bytes to AWS Rekognition for moderation analysis.
func (s *ModerationService) AnalyzeImage(
	ctx context.Context,
	imageBytes []byte,
) (*portservice.ModerationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	input := &rekognition.DetectModerationLabelsInput{
		Image: &rtypes.Image{
			Bytes: imageBytes,
		},
		MinConfidence: aws.Float32(float32(s.threshold)),
	}

	output, err := s.client.DetectModerationLabels(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("rekognition detect: %w", err)
	}

	labels := make([]media.ModerationLabel, 0, len(output.ModerationLabels))
	var maxScore float64

	for _, l := range output.ModerationLabels {
		conf := float64(aws.ToFloat32(l.Confidence))
		labels = append(labels, media.ModerationLabel{
			Name:       aws.ToString(l.Name),
			Confidence: conf,
			ParentName: aws.ToString(l.ParentName),
		})
		if conf > maxScore {
			maxScore = conf
		}
	}

	return &portservice.ModerationResult{
		Safe:   len(labels) == 0,
		Labels: labels,
		Score:  maxScore,
	}, nil
}

// AnalyzeVideo starts an async Rekognition content moderation job on a video stored in S3.
// The completion status is published to the configured SNS topic.
func (s *ModerationService) AnalyzeVideo(
	ctx context.Context,
	s3Bucket string,
	s3Key string,
) (*portservice.VideoJob, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	input := &rekognition.StartContentModerationInput{
		Video: &rtypes.Video{
			S3Object: &rtypes.S3Object{
				Bucket: aws.String(s3Bucket),
				Name:   aws.String(s3Key),
			},
		},
		MinConfidence: aws.Float32(float32(s.threshold)),
		NotificationChannel: &rtypes.NotificationChannel{
			SNSTopicArn: aws.String(s.snsTopicARN),
			RoleArn:     aws.String(s.roleARN),
		},
	}

	output, err := s.client.StartContentModeration(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("rekognition start content moderation: %w", err)
	}

	return &portservice.VideoJob{
		JobID: aws.ToString(output.JobId),
	}, nil
}

// GetVideoModerationResult fetches labels for a completed video moderation job.
func (s *ModerationService) GetVideoModerationResult(
	ctx context.Context,
	jobID string,
) (*portservice.ModerationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	labels, maxScore, err := s.collectVideoLabels(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return &portservice.ModerationResult{
		Safe:   len(labels) == 0,
		Labels: labels,
		Score:  maxScore,
	}, nil
}

// collectVideoLabels paginates through GetContentModeration results and deduplicates labels.
func (s *ModerationService) collectVideoLabels(
	ctx context.Context,
	jobID string,
) ([]media.ModerationLabel, float64, error) {
	seen := make(map[string]media.ModerationLabel)
	var maxScore float64
	var nextToken *string

	for {
		input := &rekognition.GetContentModerationInput{
			JobId:     aws.String(jobID),
			NextToken: nextToken,
		}
		output, err := s.client.GetContentModeration(ctx, input)
		if err != nil {
			return nil, 0, fmt.Errorf("rekognition get content moderation: %w", err)
		}

		if output.JobStatus != rtypes.VideoJobStatusSucceeded {
			return nil, 0, fmt.Errorf("rekognition job %s not succeeded: %s", jobID, output.JobStatus)
		}

		for _, det := range output.ModerationLabels {
			if det.ModerationLabel == nil {
				continue
			}
			conf := float64(aws.ToFloat32(det.ModerationLabel.Confidence))
			name := aws.ToString(det.ModerationLabel.Name)
			existing, ok := seen[name]
			if !ok || conf > existing.Confidence {
				seen[name] = media.ModerationLabel{
					Name:       name,
					Confidence: conf,
					ParentName: aws.ToString(det.ModerationLabel.ParentName),
				}
			}
			if conf > maxScore {
				maxScore = conf
			}
		}

		if output.NextToken == nil || aws.ToString(output.NextToken) == "" {
			break
		}
		nextToken = output.NextToken
	}

	labels := make([]media.ModerationLabel, 0, len(seen))
	for _, l := range seen {
		labels = append(labels, l)
	}
	return labels, maxScore, nil
}
