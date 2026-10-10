package entity

import (
	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
	"github.com/pgvector/pgvector-go"
)

// CardEmbedding is the embedding of one image of a Card, made by the embedding
// model named in Model. Source names the image (the hosted "catalog" image
// today). A Card has one row per model and source, so a model change adds a
// row rather than replacing one. Embedding is L2-normalised before it is
// stored. Choosing the best row per card when several match is the matcher's
// job (ticket 06), not this entity's.
type CardEmbedding struct {
	crud.BaseEntity
	CardID    uuid.UUID       `gorm:"type:uuid;not null"`
	Model     string          `gorm:"not null"`
	Source    string          `gorm:"not null"`
	Embedding pgvector.Vector `gorm:"type:vector(1536);not null"`
}

func (CardEmbedding) TableName() string { return "card_embeddings" }
