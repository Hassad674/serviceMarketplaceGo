package rekognition

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"

	"marketplace-backend/internal/domain/media"
	portservice "marketplace-backend/internal/port/service"
)

// ModerationService implements ContentModerationService using AWS Rekognition.
type ModerationService struct {
	client    *rekognition.Client
	threshold float64
}

// NewModerationService creates a Rekognition-backed moderation service.
func NewModerationService(region string, threshold float64) (*ModerationService, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("rekognition: load aws config: %w", err)
	}

	client := rekognition.NewFromConfig(cfg)
	return &ModerationService{
		client:    client,
		threshold: threshold,
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
		Image: &types.Image{
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
