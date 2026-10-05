package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
)

const logoContentType = "image/webp"

// summary counts what one run did. Matched counts Series rows whose code has a
// logo file, whether or not the row needed an update.
type summary struct {
	Matched  int
	Uploaded int
}

// logoKey names a logo's object `series/<code>.<hash8>.webp`. The hash is the
// first 8 hex characters of the content's SHA-256, so a re-cropped file gets a
// new key and the one-year immutable cache stays safe (docs/adr/0017).
func logoKey(code string, content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("series/%s.%s.webp", code, hex.EncodeToString(sum[:4]))
}

// hostLogos uploads every <code>.webp in dir and stores its key on the Series
// rows with that code. A row already holding the file's key is left alone, so a
// re-run with unchanged files uploads nothing. A failure on one file is logged
// and the rest still run; the returned error joins them all, including a code
// that matched no row (a wrong file name must not pass silently).
func hostLogos(ctx context.Context, dir string, series crud.Repository[entity.Series], store corestore.ObjectStore) (summary, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.webp"))
	if err != nil {
		return summary{}, ungerr.Wrapf(err, "listing %s", dir)
	}
	if len(files) == 0 {
		return summary{}, ungerr.Unknownf("no .webp files in %s", dir)
	}

	var sum summary
	var errs error
	for _, file := range files {
		code := strings.TrimSuffix(filepath.Base(file), ".webp")
		matched, uploaded, err := hostLogo(ctx, file, code, series, store)
		sum.Matched += matched
		if uploaded {
			sum.Uploaded++
		}
		if err != nil {
			logger.Error(err)
			errs = errors.Join(errs, err)
		}
	}
	return sum, errs
}

func hostLogo(ctx context.Context, file, code string, series crud.Repository[entity.Series], store corestore.ObjectStore) (matched int, uploaded bool, err error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return 0, false, ungerr.Wrapf(err, "reading %s", file)
	}

	rows, err := series.FindAll(ctx, crud.Specification[entity.Series]{Model: entity.Series{Code: code}})
	if err != nil {
		return 0, false, err
	}
	if len(rows) == 0 {
		return 0, false, ungerr.Unknownf("series %q: no row matched (does the file name equal the series code?)", code)
	}
	logger.Infof("series %q: matched %d row(s)", code, len(rows))

	key := logoKey(code, content)
	var stale []entity.Series
	for _, row := range rows {
		if row.ImageKey != key {
			stale = append(stale, row)
		}
	}
	if len(stale) == 0 {
		logger.Infof("series %q: up to date (%s)", code, key)
		return len(rows), false, nil
	}

	if err := store.Put(ctx, key, logoContentType, content); err != nil {
		return len(rows), false, fmt.Errorf("series %q: uploading %s: %w", code, key, err)
	}
	for _, row := range stale {
		row.ImageKey = key
		if _, err := series.Update(ctx, row); err != nil {
			return len(rows), true, fmt.Errorf("series %q: saving key %s: %w", code, key, err)
		}
	}
	logger.Infof("series %q: hosted %s", code, key)
	return len(rows), true, nil
}
