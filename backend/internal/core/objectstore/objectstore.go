// Package objectstore defines the seam between the ingester and the bucket
// that hosts card and Expansion Set images (docs/adr/0016). The production
// adapter is adapters/objectstore.R2Store; tests use the generated mock of
// ObjectStore.
package objectstore

import "context"

// ObjectStore stores immutable image originals under deterministic keys.
type ObjectStore interface {
	// Put writes body under key with the given content type, replacing any
	// existing object, so a retry after a partial failure is safe.
	Put(ctx context.Context, key, contentType string, body []byte) error
}
