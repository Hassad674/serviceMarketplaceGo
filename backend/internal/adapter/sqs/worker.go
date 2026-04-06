package sqs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// JobFinalizer is implemented by a service able to complete a video moderation
// job given its Rekognition JobId (typically the media application service).
type JobFinalizer interface {
	FinalizeVideoJob(ctx context.Context, jobID string) error
}

// WorkerDeps groups the dependencies required to build a Worker.
type WorkerDeps struct {
	Region    string
	QueueURL  string
	Finalizer JobFinalizer
}

// Worker polls an SQS queue for Rekognition completion notifications and
// dispatches each one to the JobFinalizer. Designed to run as a single
// long-lived goroutine per instance.
type Worker struct {
	client    *sqs.Client
	queueURL  string
	finalizer JobFinalizer
}

// NewWorker constructs an SQS worker using the default AWS credential chain.
func NewWorker(deps WorkerDeps) (*Worker, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(deps.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("sqs worker: load aws config: %w", err)
	}
	return &Worker{
		client:    sqs.NewFromConfig(cfg),
		queueURL:  deps.QueueURL,
		finalizer: deps.Finalizer,
	}, nil
}

// Start runs the polling loop until ctx is cancelled. It logs and recovers
// from transient errors so it can run unattended.
func (w *Worker) Start(ctx context.Context) {
	slog.Info("sqs worker started", "queue_url", w.queueURL)
	for {
		if err := ctx.Err(); err != nil {
			slog.Info("sqs worker stopped", "reason", err)
			return
		}
		w.pollOnce(ctx)
	}
}

// pollOnce receives up to 10 messages and processes them. Long polling keeps
// the request alive for up to 20s, so this naturally throttles when idle.
func (w *Worker) pollOnce(ctx context.Context) {
	out, err := w.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(w.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
		VisibilityTimeout:   60,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		slog.Error("sqs worker: receive", "error", err)
		// Small backoff so we don't hammer the API on persistent failures.
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
		return
	}

	for _, msg := range out.Messages {
		w.handleMessage(ctx, msg.Body, msg.ReceiptHandle)
	}
}

// handleMessage parses one SQS message, finalizes the job and deletes the
// message on success. On failure the message is left in the queue so it will
// be retried (and eventually moved to a DLQ if configured).
func (w *Worker) handleMessage(ctx context.Context, body *string, receipt *string) {
	if body == nil || receipt == nil {
		return
	}
	jobID, status, err := parseNotification(*body)
	if err != nil {
		slog.Error("sqs worker: parse message", "error", err, "body", *body)
		w.deleteMessage(ctx, receipt)
		return
	}
	if status != "SUCCEEDED" {
		slog.Warn("sqs worker: job did not succeed", "job_id", jobID, "status", status)
		w.deleteMessage(ctx, receipt)
		return
	}
	if jobID == "" {
		slog.Warn("sqs worker: missing job id", "body", *body)
		w.deleteMessage(ctx, receipt)
		return
	}

	finalizeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := w.finalizer.FinalizeVideoJob(finalizeCtx, jobID); err != nil {
		slog.Error("sqs worker: finalize job", "error", err, "job_id", jobID)
		return
	}
	slog.Info("sqs worker: job finalized", "job_id", jobID)
	w.deleteMessage(ctx, receipt)
}

func (w *Worker) deleteMessage(ctx context.Context, receipt *string) {
	_, err := w.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(w.queueURL),
		ReceiptHandle: receipt,
	})
	if err != nil {
		slog.Warn("sqs worker: delete message", "error", err)
	}
}

// rekognitionNotification matches the payload that Rekognition publishes to
// SNS when a content moderation job finishes.
type rekognitionNotification struct {
	JobID  string `json:"JobId"`
	Status string `json:"Status"`
	API    string `json:"API"`
}

// snsEnvelope matches the outer envelope used when the SNS -> SQS delivery
// is NOT configured for raw delivery.
type snsEnvelope struct {
	Type    string `json:"Type"`
	Message string `json:"Message"`
}

// parseNotification handles both raw and enveloped SNS payloads.
func parseNotification(body string) (jobID, status string, err error) {
	// Try raw delivery first.
	var raw rekognitionNotification
	if jsonErr := json.Unmarshal([]byte(body), &raw); jsonErr == nil && raw.JobID != "" {
		return raw.JobID, raw.Status, nil
	}
	// Fall back to the SNS envelope.
	var env snsEnvelope
	if jsonErr := json.Unmarshal([]byte(body), &env); jsonErr != nil {
		return "", "", fmt.Errorf("unmarshal sqs body: %w", jsonErr)
	}
	if env.Message == "" {
		return "", "", errors.New("empty SNS message body")
	}
	var inner rekognitionNotification
	if jsonErr := json.Unmarshal([]byte(env.Message), &inner); jsonErr != nil {
		return "", "", fmt.Errorf("unmarshal sns message: %w", jsonErr)
	}
	return inner.JobID, inner.Status, nil
}
