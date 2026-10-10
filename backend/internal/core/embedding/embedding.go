// Package embedding defines the seams between the card embedding batch job and
// the outside world: the embedding provider and the image host. The production
// adapters live in adapters/embedding; tests use the generated mocks.
package embedding

import (
	"context"
)

// Dimensions is the vector length asked of the provider. 1536 is the largest
// size Gemini recommends that stays under pgvector's 2000-dimension index limit
// (ADR-0018).
const Dimensions = 1536

// Image is one card image ready to embed. Key is the caller's identity for the
// image (the card's ID), echoed back with its vector or failure.
type Image struct {
	Key      string
	MIMEType string
	Data     []byte
}

// JobStatus is where a batch job stands at the provider, reduced to what the
// caller acts on.
type JobStatus string

const (
	// JobRunning is a job not yet finished (pending, queued, running or paused).
	JobRunning JobStatus = "running"
	// JobSucceeded is a finished job whose results are in Outcome.
	JobSucceeded JobStatus = "succeeded"
	// JobFailed is a job that failed, was cancelled or expired, and so has no
	// results.
	JobFailed JobStatus = "failed"
)

// Outcome is what Collect reports for a job. Vectors and Failures are set only
// once the job has succeeded.
type Outcome struct {
	Status JobStatus
	// Reason says why a failed job failed.
	Reason string
	// Vectors holds the unit-length vectors of the images that embedded, keyed by
	// Image.Key.
	Vectors map[string][]float32
	// Failures holds the error of each image that did not embed, keyed by
	// Image.Key.
	Failures map[string]string
}

// BatchEmbedder embeds images through a provider's asynchronous batch API,
// where a job can take up to a day. Submit starts a job and returns its name;
// Collect reports the job's status and, once it has succeeded, its vectors.
type BatchEmbedder interface {
	Submit(ctx context.Context, images []Image) (string, error)
	Collect(ctx context.Context, job string) (Outcome, error)
}

// ImageFetcher downloads one hosted image by its full address and returns it as
// a PNG or JPEG, the types the provider accepts.
type ImageFetcher interface {
	Fetch(ctx context.Context, url string) (mimeType string, data []byte, err error)
}
