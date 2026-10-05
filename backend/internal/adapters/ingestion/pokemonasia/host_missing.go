package pokemonasia

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"gorm.io/gorm"
)

// hostMissingBatchSize bounds how many card rows are held in memory at once.
const hostMissingBatchSize = 200

// hostMissingWhere selects rows that have a source address but no hosted key.
const hostMissingWhere = "image_key = '' AND source_image_url <> ''"

// HostMissingImages copies the image of every Expansion Set and card that has
// a source address but no hosted key into the object store, without crawling
// the source site. It is the backfill for rows ingested before hosting existed,
// and it shares the ingester's download rules and failure handling: a failed
// row keeps an empty key, is recorded in Summary.Failures, and does not abort
// the run, so re-running retries it. A non-empty setFilter limits the run to
// the Expansion Set with that site code. It needs an object store.
func (in *Ingester) HostMissingImages(ctx context.Context, setFilter string) (Summary, error) {
	if in.store == nil {
		return Summary{}, errors.New("an object store is required to host images (set R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET)")
	}
	logger.Infof("hosting missing images (set filter: %q)", setFilter)

	// The hosted counter and failure list accumulate on the Ingester, so the
	// summary reports only what changed during this call.
	hostedBefore := in.imagesHosted.Load()
	in.failuresMu.Lock()
	failuresBefore := len(in.failures)
	in.failuresMu.Unlock()

	sets, err := in.listSets(ctx, setFilter)
	if err != nil {
		return Summary{}, err
	}
	setCodes := make(map[uuid.UUID]string, len(sets))
	setIDs := make([]uuid.UUID, 0, len(sets))
	for i, set := range sets {
		setCodes[set.ID] = set.Code
		setIDs = append(setIDs, set.ID)
		before := in.imagesHosted.Load()
		in.hostSetCover(ctx, set, set.Code)
		if in.imagesHosted.Load() > before {
			logger.Infof("[%d/%d] set %s: cover hosted", i+1, len(sets), set.Code)
		} else {
			logger.Infof("[%d/%d] set %s: no cover hosted (already hosted, no source address, or failed)", i+1, len(sets), set.Code)
		}
	}

	if err := in.hostMissingCards(ctx, setIDs, setCodes); err != nil {
		return Summary{}, err
	}

	summary := Summary{ImagesHosted: int(in.imagesHosted.Load() - hostedBefore)}
	in.failuresMu.Lock()
	summary.Failures = append([]IngestFailure(nil), in.failures[failuresBefore:]...)
	in.failuresMu.Unlock()
	logger.Infof("hosting finished: %d image(s) hosted, %d failure(s)", summary.ImagesHosted, len(summary.Failures))
	return summary, ctx.Err()
}

// listSets returns every Expansion Set (or just the one with setFilter's code).
func (in *Ingester) listSets(ctx context.Context, setFilter string) ([]entity.ExpansionSet, error) {
	db, err := in.sets.GetGormInstance(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting database handle: %w", err)
	}
	query := db.WithContext(ctx)
	if setFilter != "" {
		query = query.Where("code = ?", setFilter)
	}
	var sets []entity.ExpansionSet
	if err := query.Order("id").Find(&sets).Error; err != nil {
		return nil, fmt.Errorf("listing expansion sets: %w", err)
	}
	return sets, nil
}

// hostMissingCards hosts the unhosted cards of the given Expansion Sets in id
// order, one batch at a time. The id cursor means a card that fails to host is
// not fetched again within this run.
func (in *Ingester) hostMissingCards(ctx context.Context, setIDs []uuid.UUID, setCodes map[uuid.UUID]string) error {
	if len(setIDs) == 0 {
		return nil
	}
	db, err := in.cards.GetGormInstance(ctx)
	if err != nil {
		return fmt.Errorf("getting database handle: %w", err)
	}

	var total int64
	if err := db.WithContext(ctx).Model(&entity.Card{}).
		Where(hostMissingWhere+" AND expansion_set_id IN ?", setIDs).
		Count(&total).Error; err != nil {
		return fmt.Errorf("counting cards without a hosted image: %w", err)
	}
	logger.Infof("%d card(s) to host", total)

	done := 0
	cursor := uuid.Nil
	for ctx.Err() == nil {
		var batch []entity.Card
		err := db.WithContext(ctx).
			Where(hostMissingWhere+" AND expansion_set_id IN ? AND id > ?", setIDs, cursor).
			Order("id").Limit(hostMissingBatchSize).
			Find(&batch).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("listing cards without a hosted image: %w", err)
		}
		if len(batch) == 0 {
			return nil
		}
		for _, card := range batch {
			in.hostCardImage(ctx, card, setCodes[card.ExpansionSetID], card.LocalID)
		}
		cursor = batch[len(batch)-1].ID
		done += len(batch)
		logger.Infof("processed %d/%d card(s)", done, total)
	}
	return nil
}
