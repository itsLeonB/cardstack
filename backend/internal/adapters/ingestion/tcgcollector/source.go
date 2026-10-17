package tcgcollector

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsLeonB/ungerr"
)

// PageSource returns the HTML of a TCG Collector page by its URL.
type PageSource interface {
	Fetch(ctx context.Context, pageURL string) ([]byte, error)
}

// dirSource reads pages saved by hand from one directory. The page at
// https://www.tcgcollector.com/sets/id is read from sets-id.html.
type dirSource struct {
	dir string
}

// NewDirSource returns a PageSource that reads saved pages from dir.
func NewDirSource(dir string) PageSource {
	return dirSource{dir: dir}
}

func (s dirSource) Fetch(_ context.Context, pageURL string) ([]byte, error) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return nil, ungerr.Wrapf(err, "parsing page URL %s", pageURL)
	}

	name := strings.ReplaceAll(strings.Trim(parsed.Path, "/"), "/", "-") + ".html"
	page, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return nil, ungerr.Wrapf(err, "reading saved page %s", name)
	}

	return page, nil
}
