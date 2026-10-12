// Package objectstore defines the seam between the API and ingesters and the
// bucket that hosts card and Expansion Set images (docs/adr/0016). The
// production adapter is adapters/objectstore.R2Store; tests use the generated
// mock of ObjectStore.
package objectstore

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is what ReadHead reports for a key with no object.
var ErrNotFound = errors.New("object not found")

// ObjectStore stores immutable image originals under deterministic keys.
type ObjectStore interface {
	// Put writes body under key with the given content type, replacing any
	// existing object, so a retry after a partial failure is safe.
	Put(ctx context.Context, key, contentType string, body []byte) error
	// Get reads the object stored under key. It fails, rather than
	// truncating, when the object exceeds the store's size cap.
	Get(ctx context.Context, key string) ([]byte, error)
	// PresignPut returns an address that accepts one PUT of an object under key
	// with exactly the given content type until it expires ttl from now, and
	// that moment (docs/adr/0019).
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, time.Time, error)
	// ReadHead reads at most the first n bytes of the object under key. It
	// returns ErrNotFound when there is no such object.
	ReadHead(ctx context.Context, key string, n int) ([]byte, error)
	// Delete removes the object under key; a missing object is not an error.
	Delete(ctx context.Context, key string) error
}
