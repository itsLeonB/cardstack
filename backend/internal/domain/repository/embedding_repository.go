package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmbeddingRepository stores card image embeddings and answers the queries the
// embedding batch job and the matcher need. Every query takes a model, because
// a query only ever compares rows of one model.
type EmbeddingRepository interface {
	// Upsert stores embeddings, replacing any row with the same card, model and
	// source. An empty slice is a no-op.
	Upsert(ctx context.Context, embeddings []entity.CardEmbedding) error
	// CountEmbeddingStates counts the cards of setCode ("" for every set) by
	// their state for model and source.
	CountEmbeddingStates(ctx context.Context, model, source, setCode string) (EmbeddingStates, error)
	// ListPendingCards returns the cards of setCode that have a hosted image, no
	// embedding for model and source, and no place in an outstanding batch for
	// them, ordered by set code and local ID. Only ID and ImageKey are set on
	// each card.
	ListPendingCards(ctx context.Context, model, source, setCode string) ([]entity.Card, error)
	// CreateBatch records a submitted batch and the cards it covers in one
	// transaction. It sets batch.ID.
	CreateBatch(ctx context.Context, batch *entity.EmbeddingBatch, cardIDs []uuid.UUID) error
	// ListSubmittedBatches returns the batches still awaiting collection.
	ListSubmittedBatches(ctx context.Context) ([]entity.EmbeddingBatch, error)
	// SetBatchState moves a batch to state, stamping collected_at when it is
	// collected.
	SetBatchState(ctx context.Context, id uuid.UUID, state entity.EmbeddingBatchState) error
	// NearestCards returns the limit rows whose model embedding is closest to
	// query by cosine distance, nearest first. A card with several rows for the
	// model appears once per row; choosing the best row per card is the
	// matcher's job (ticket 06).
	NearestCards(ctx context.Context, model string, query []float32, limit int) ([]CardDistance, error)
}

// EmbeddingStates counts cards of one scope by their embedding state for one model.
type EmbeddingStates struct {
	Total    int64
	NoImage  int64
	Embedded int64
	InFlight int64
}

// CardDistance is the ID of a card and its cosine distance to a query vector.
type CardDistance struct {
	CardID   uuid.UUID
	Distance float64
}

// embeddingRepository is EmbeddingRepository's GORM-backed implementation. It
// embeds a crud.Repository only for GetGormInstance, so every method runs in
// the transaction carried by ctx, if any.
type embeddingRepository struct {
	crud.Repository[entity.CardEmbedding]
}

// NewEmbeddingRepository builds an EmbeddingRepository over base's database.
func NewEmbeddingRepository(base crud.Repository[entity.CardEmbedding]) EmbeddingRepository {
	return &embeddingRepository{Repository: base}
}

func (r *embeddingRepository) Upsert(ctx context.Context, embeddings []entity.CardEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	err = db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "card_id"}, {Name: "model"}, {Name: "source"}},
			DoUpdates: clause.AssignmentColumns([]string{"embedding", "updated_at"}),
		}).
		Create(&embeddings).
		Error
	if err != nil {
		return ungerr.Wrap(err, "upserting card embeddings")
	}
	return nil
}

func (r *embeddingRepository) CountEmbeddingStates(ctx context.Context, model, source, setCode string) (EmbeddingStates, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return EmbeddingStates{}, err
	}

	var states EmbeddingStates
	err = cardsOfSet(db, setCode).
		Select(`COUNT(*) AS total,
			COUNT(*) FILTER (WHERE cards.image_key = '') AS no_image,
			COUNT(*) FILTER (WHERE cards.image_key <> '' AND `+embeddedSQL+`) AS embedded,
			COUNT(*) FILTER (WHERE cards.image_key <> '' AND `+inFlightSQL+`) AS in_flight`,
			model, source, entity.EmbeddingBatchSubmitted, model, source).
		Scan(&states).
		Error
	if err != nil {
		return EmbeddingStates{}, ungerr.Wrap(err, "counting card embedding states")
	}
	return states, nil
}

func (r *embeddingRepository) ListPendingCards(ctx context.Context, model, source, setCode string) ([]entity.Card, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var cards []entity.Card
	err = cardsOfSet(db, setCode).
		Select("cards.id, cards.image_key").
		Where("cards.image_key <> ''").
		Where("NOT "+embeddedSQL, model, source).
		Where("NOT "+inFlightSQL, entity.EmbeddingBatchSubmitted, model, source).
		Order("expansion_sets.code, cards.local_id").
		Find(&cards).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing cards pending embedding")
	}
	return cards, nil
}

func (r *embeddingRepository) NearestCards(ctx context.Context, model string, query []float32, limit int) ([]CardDistance, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	vector := pgvector.NewVector(query)
	var nearest []CardDistance
	err = db.
		Table("card_embeddings").
		Select("card_id, embedding <=> ? AS distance", vector).
		Where("model = ?", model).
		Order(clause.OrderBy{Expression: clause.Expr{SQL: "embedding <=> ?", Vars: []any{vector}}}).
		Limit(limit).
		Scan(&nearest).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "searching nearest card embeddings")
	}
	return nearest, nil
}

// cardsOfSet scopes a query to the cards of one Expansion Set by its code, or
// to every card when setCode is "".
func cardsOfSet(db *gorm.DB, setCode string) *gorm.DB {
	query := db.Model(&entity.Card{}).Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id")
	if setCode != "" {
		query = query.Where("expansion_sets.code = ?", setCode)
	}
	return query
}

// embeddedSQL matches a card that already has an embedding for a model and
// source. inFlightSQL matches a card in a batch in the given state for them
// (args: state, model, source). Only a submitted batch makes a card not
// pending until the batch is collected or failed.
const (
	embeddedSQL = "EXISTS (SELECT 1 FROM card_embeddings ce WHERE ce.card_id = cards.id AND ce.model = ? AND ce.source = ?)"
	inFlightSQL = "cards.id IN (SELECT ebc.card_id FROM embedding_batch_cards ebc JOIN embedding_batches eb ON eb.id = ebc.batch_id WHERE eb.state = ? AND eb.model = ? AND eb.source = ?)"
)

func (r *embeddingRepository) CreateBatch(ctx context.Context, batch *entity.EmbeddingBatch, cardIDs []uuid.UUID) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(batch).Error; err != nil {
			return err
		}
		if len(cardIDs) == 0 {
			return nil
		}
		links := make([]entity.EmbeddingBatchCard, len(cardIDs))
		for i, id := range cardIDs {
			links[i] = entity.EmbeddingBatchCard{BatchID: batch.ID, CardID: id}
		}
		return tx.Create(&links).Error
	})
	if err != nil {
		return ungerr.Wrap(err, "recording embedding batch")
	}
	return nil
}

func (r *embeddingRepository) ListSubmittedBatches(ctx context.Context) ([]entity.EmbeddingBatch, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var batches []entity.EmbeddingBatch
	err = db.
		Where("state = ?", entity.EmbeddingBatchSubmitted).
		Order("created_at").
		Find(&batches).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing submitted embedding batches")
	}
	return batches, nil
}

func (r *embeddingRepository) SetBatchState(ctx context.Context, id uuid.UUID, state entity.EmbeddingBatchState) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	updates := map[string]any{"state": state, "updated_at": gorm.Expr("now()")}
	if state == entity.EmbeddingBatchCollected {
		updates["collected_at"] = gorm.Expr("now()")
	}
	err = db.
		Model(&entity.EmbeddingBatch{}).
		Where("id = ?", id).
		Updates(updates).
		Error
	if err != nil {
		return ungerr.Wrap(err, "setting embedding batch state")
	}
	return nil
}
