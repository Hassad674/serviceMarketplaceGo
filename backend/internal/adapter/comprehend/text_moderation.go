package comprehend

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/comprehend"
	ctypes "github.com/aws/aws-sdk-go-v2/service/comprehend/types"

	portservice "marketplace-backend/internal/port/service"
)

// TextModerationService implements port/service.TextModerationService using AWS Comprehend.
type TextModerationService struct {
	client    *comprehend.Client
	threshold float64
}

// NewTextModerationService creates a Comprehend-backed text moderation service.
// threshold is the score (0-1) at or above which text is considered unsafe.
func NewTextModerationService(region string) (*TextModerationService, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("comprehend: load aws config: %w", err)
	}

	client := comprehend.NewFromConfig(cfg)
	return &TextModerationService{
		client:    client,
		threshold: 0.5,
	}, nil
}

// AnalyzeText sends text to AWS Comprehend Toxicity Detection and returns moderation results.
func (s *TextModerationService) AnalyzeText(
	ctx context.Context,
	text string,
) (*portservice.TextModerationResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Comprehend accepts up to 10 segments of 1KB each.
	// Truncate to 1KB if necessary (single segment is sufficient for messages).
	if len(text) > 1000 {
		text = text[:1000]
	}

	input := &comprehend.DetectToxicContentInput{
		LanguageCode: ctypes.LanguageCodeEn,
		TextSegments: []ctypes.TextSegment{
			{Text: aws.String(text)},
		},
	}

	output, err := s.client.DetectToxicContent(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("comprehend detect toxic content: %w", err)
	}

	return mapComprehendResult(output), nil
}

// mapComprehendResult converts AWS Comprehend response to our port result type.
func mapComprehendResult(output *comprehend.DetectToxicContentOutput) *portservice.TextModerationResult {
	var labels []portservice.TextModerationLabel
	var maxScore float64

	for _, result := range output.ResultList {
		// Use the overall toxicity score if available.
		if result.Toxicity != nil {
			score := float64(aws.ToFloat32(result.Toxicity))
			if score > maxScore {
				maxScore = score
			}
		}

		for _, label := range result.Labels {
			score := float64(aws.ToFloat32(label.Score))
			labels = append(labels, portservice.TextModerationLabel{
				Name:  string(label.Name),
				Score: score,
			})
			if score > maxScore {
				maxScore = score
			}
		}
	}

	return &portservice.TextModerationResult{
		Labels:   labels,
		MaxScore: maxScore,
		IsSafe:   maxScore < 0.5,
	}
}
