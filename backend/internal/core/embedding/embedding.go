// Package embedding defines the seams between the card embedding batch job and
// the outside world: the embedding provider and the image host. The production
// adapters live in adapters/embedding; tests use the generated mocks.
package embedding

import (
	"context"
	"errors"
)

// Embedder turns one image into an embedding vector. The vector comes back
// L2-normalised, so cosine distance and dot product rank the same.
type Embedder interface {
	Embed(ctx context.Context, image []byte) ([]float32, error)
}

// ImageFetcher downloads one hosted image by its full address.
type ImageFetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// ErrDailyQuotaExhausted means the provider refused further requests for the
// day. Retrying within the same day cannot succeed, so the batch job stops
// rather than failing card by card.
var ErrDailyQuotaExhausted = errors.New("embedding provider daily quota exhausted")
