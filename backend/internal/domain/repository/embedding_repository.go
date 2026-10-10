package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmbeddingRepository stores card image embeddings and answers the queries the
// embedding batch job and the matcher need. Every query takes a model, because
// a query only ever compares rows of one model.
type EmbeddingRepository interface {
	// Upsert stores e for its card, replacing whatever embedding the card had.
	Upsert(ctx context.Context, e entity.CardEmbedding) error
	// CountEmbeddingStates counts the cards of setCode ("" for every set) by
	// their state for model.
	CountEmbeddingStates(ctx context.Context, model, setCode string) (EmbeddingStates, error)
	// ListPendingCards returns the cards of setCode that have a hosted image
	// and no embedding from model, ordered by set code and local ID. Only ID
	// and ImageKey are set on each card.
	ListPendingCards(ctx context.Context, model, setCode string) ([]entity.Card, error)
	// NearestCards returns the limit cards whose model embedding is closest to
	// query by cosine distance, nearest first.
	NearestCards(ctx context.Context, model string, query []float32, limit int) ([]CardDistance, error)
}

// EmbeddingStates counts cards of one scope by their embedding state for one model.
type EmbeddingStates struct {
	Total    int64
	NoImage  int64
	Embedded int64
}

// CardDistance is a Card and its cosine distance to a query vector.
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

func (r *embeddingRepository) Upsert(ctx context.Context, e entity.CardEmbedding) error {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return err
	}

	return db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "card_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"model", "embedding", "updated_at"}),
		}).
		Create(&e).
		Error
}

func (r *embeddingRepository) CountEmbeddingStates(ctx context.Context, model, setCode string) (EmbeddingStates, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return EmbeddingStates{}, err
	}

	var states EmbeddingStates
	err = cardsOfSet(db, setCode).
		Select(`COUNT(*) AS total,
			COUNT(*) FILTER (WHERE cards.image_key = '') AS no_image,
			COUNT(*) FILTER (WHERE cards.image_key <> '' AND EXISTS (
				SELECT 1 FROM card_embeddings ce WHERE ce.card_id = cards.id AND ce.model = ?
			)) AS embedded`, model).
		Scan(&states).
		Error
	return states, err
}

func (r *embeddingRepository) ListPendingCards(ctx context.Context, model, setCode string) ([]entity.Card, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	var cards []entity.Card
	err = cardsOfSet(db, setCode).
		Select("cards.id, cards.image_key").
		Where("cards.image_key <> ''").
		Where("NOT EXISTS (SELECT 1 FROM card_embeddings ce WHERE ce.card_id = cards.id AND ce.model = ?)", model).
		Order("expansion_sets.code, cards.local_id").
		Find(&cards).
		Error
	return cards, err
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
	return nearest, err
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
