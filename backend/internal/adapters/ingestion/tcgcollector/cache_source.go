package tcgcollector

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/itsLeonB/ungerr"
)

// cacheSource keeps every page it fetches in dir, so a later run reads from
// disk and needs no requests. A run with -refresh (withRefresh) refetches and
// overwrites them.
type cacheSource struct {
	dir  string
	next PageSource
}

// NewCacheSource wraps next with an on-disk page cache in dir.
func NewCacheSource(dir string, next PageSource) PageSource {
	return cacheSource{dir: dir, next: next}
}

type refreshKey struct{}

// withRefresh asks the cache to refetch the pages this context fetches.
func withRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, refreshKey{}, true)
}

func (c cacheSource) Fetch(ctx context.Context, pageURL string) ([]byte, error) {
	name, err := pageFileName(pageURL)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(c.dir, name)

	if !refreshRequested(ctx) {
		page, err := os.ReadFile(path)
		if err == nil {
			return page, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, ungerr.Wrapf(err, "reading cached page %s", name)
		}
	}

	page, err := c.next.Fetch(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	// Write under a temporary name first: a write cut short must not leave a
	// truncated page that later runs would parse as the real one.
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return nil, ungerr.Wrapf(err, "creating page cache %s", c.dir)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, page, 0o644); err != nil {
		return nil, ungerr.Wrapf(err, "caching page %s", name)
	}
	if err := os.Rename(tmp, path); err != nil {
		return nil, ungerr.Wrapf(err, "caching page %s", name)
	}

	return page, nil
}

func refreshRequested(ctx context.Context) bool {
	refresh, _ := ctx.Value(refreshKey{}).(bool)
	return refresh
}
