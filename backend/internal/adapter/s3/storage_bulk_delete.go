package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	portservice "marketplace-backend/internal/port/service"
)

// s3BulkDeleteChunkSize is the maximum number of objects S3 / R2 /
// MinIO accept in a single DeleteObjects request. The S3 API caps it
// at 1000; we batch larger purges into multiple round-trips.
const s3BulkDeleteChunkSize = 1000

// BulkDelete deletes every key in `keys` using the S3-compatible
// DeleteObjects API. Returns one BulkDeleteResult per requested key,
// in the same order, with Err non-nil for failures.
//
// Best-effort semantics: the function never aborts the batch on the
// first failure. Per-object errors reported by the server are mapped
// onto their key's BulkDeleteResult.Err. The function only returns a
// top-level error when the entire batch could not be issued (e.g. the
// SDK call returned a transport error before any key was processed).
//
// Used by the GDPR right-to-erasure cron to purge a user's uploaded
// media before the SQL anonymization step. The caller persists the
// results in storage_purge_audits as compliance evidence.
func (s *StorageService) BulkDelete(ctx context.Context, keys []string) ([]portservice.BulkDeleteResult, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	results := make([]portservice.BulkDeleteResult, len(keys))
	for i, k := range keys {
		results[i] = portservice.BulkDeleteResult{Key: k}
	}

	// Index keys by position so the reply (which may reorder) can be
	// stitched back into the caller's input order.
	indexByKey := make(map[string]int, len(keys))
	for i, k := range keys {
		indexByKey[k] = i
	}

	for start := 0; start < len(keys); start += s3BulkDeleteChunkSize {
		end := start + s3BulkDeleteChunkSize
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[start:end]

		if err := s.deleteChunk(ctx, chunk, results, indexByKey); err != nil {
			return results, err
		}
	}
	return results, nil
}

// deleteChunk issues one DeleteObjects request for up to
// s3BulkDeleteChunkSize keys and writes the per-key outcomes back into
// results. A transport-level failure (the request itself failed)
// returns an error AND marks every key in the chunk as failed so the
// caller still has per-key audit material.
func (s *StorageService) deleteChunk(
	ctx context.Context,
	chunk []string,
	results []portservice.BulkDeleteResult,
	indexByKey map[string]int,
) error {
	objects := make([]types.ObjectIdentifier, len(chunk))
	for i, k := range chunk {
		key := k // bind for pointer
		objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
	}

	out, err := s.client.DeleteObjects(ctx, &awss3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false), // we want per-key Deleted/Errors entries
		},
	})
	if err != nil {
		// Transport / authorization failure on the whole batch:
		// every key in this chunk is unknown-state. Mark them all
		// failed so the audit captures the situation, then bubble.
		for _, k := range chunk {
			if idx, ok := indexByKey[k]; ok {
				results[idx].Err = fmt.Errorf("s3 bulk delete batch: %w", err)
			}
		}
		return fmt.Errorf("s3 bulk delete: %w", err)
	}

	// Surface per-object errors reported by the server.
	for _, e := range out.Errors {
		if e.Key == nil {
			continue
		}
		idx, ok := indexByKey[*e.Key]
		if !ok {
			continue
		}
		code := ""
		if e.Code != nil {
			code = *e.Code
		}
		message := ""
		if e.Message != nil {
			message = *e.Message
		}
		results[idx].Err = errors.New("s3 bulk delete (" + code + "): " + message)
	}
	return nil
}
