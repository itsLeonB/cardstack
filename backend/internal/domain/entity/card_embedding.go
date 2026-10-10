package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// CardEmbedding is the embedding of one Card's hosted image, made by the
// embedding model named in Model. It is keyed by Card: re-embedding replaces
// the row, whichever model made the old one. Embedding is L2-normalised by the
// embedding adapter before it is stored.
type CardEmbedding struct {
	CardID    uuid.UUID       `gorm:"type:uuid;primaryKey"`
	Model     string          `gorm:"not null"`
	Embedding pgvector.Vector `gorm:"type:vector(1536);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (CardEmbedding) TableName() string { return "card_embeddings" }
