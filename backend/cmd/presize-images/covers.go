package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	corestore "github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

const coverContentType = "image/webp"

// resizeFunc turns a hosted original into the cover bytes. vips does it in
// production; tests fake it (docs/adr/0011).
type resizeFunc func(ctx context.Context, original []byte) ([]byte, error)

type summary struct {
	Selected int
	Resized  int
	Failed   int
}

// coverKey names a cover `expansion-sets/<id>.<hash8>.webp`. The hash is the
// first 8 hex characters of the SHA-256 of the output bytes, so a changed size
// gets a new key and the one-year immutable cache stays safe (docs/adr/0017).
func coverKey(setID string, content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("expansion-sets/%s.%s.webp", setID, hex.EncodeToString(sum[:4]))
}

// presizeCovers resizes the original of every Expansion Set that has one but no
// cover, uploads the result and stores its key in cover_key. A non-empty
// setCode limits it to that set. A set that already has a cover is not
// selected, so a re-run does nothing. A failure on one set is logged and the
// set keeps an empty cover for the next run; the returned error joins them all.
func presizeCovers(ctx context.Context, sets repository.ExpansionSetRepository, store corestore.ObjectStore, resize resizeFunc, setCode string) (summary, error) {
	rows, err := sets.ListMissingCovers(ctx, setCode)
	if err != nil {
		return summary{}, err
	}

	sum := summary{Selected: len(rows)}
	var errs error
	for i, row := range rows {
		key, err := presizeCover(ctx, row, sets, store, resize)
		if err != nil {
			sum.Failed++
			err = fmt.Errorf("set %s: %w", row.Code, err)
			logger.Errorf("[%d/%d] %v (left for the next run)", i+1, len(rows), err)
			errs = errors.Join(errs, err)
			continue
		}
		sum.Resized++
		logger.Infof("[%d/%d] set %s: cover %s", i+1, len(rows), row.Code, key)
	}
	return sum, errs
}

func presizeCover(ctx context.Context, row entity.ExpansionSet, sets repository.ExpansionSetRepository, store corestore.ObjectStore, resize resizeFunc) (string, error) {
	original, err := store.Get(ctx, row.ImageKey)
	if err != nil {
		return "", err
	}
	cover, err := resize(ctx, original)
	if err != nil {
		return "", err
	}
	key := coverKey(row.ID.String(), cover)
	if err := store.Put(ctx, key, coverContentType, cover); err != nil {
		return "", err
	}
	row.CoverKey = key
	if _, err := sets.Update(ctx, row); err != nil {
		return "", err
	}
	return key, nil
}
