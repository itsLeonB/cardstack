package entity

import (
	"time"

	"github.com/google/uuid"
	crud "github.com/itsLeonB/go-crud"
)

// EmbeddingBatchState is where a batch job stands in this app's bookkeeping,
// which is separate from the job's own state at Gemini.
type EmbeddingBatchState string

const (
	// EmbeddingBatchSubmitted is a job still awaiting collection. Its cards are
	// not pending for the same model and source.
	EmbeddingBatchSubmitted EmbeddingBatchState = "submitted"
	// EmbeddingBatchCollected is a job whose vectors are stored.
	EmbeddingBatchCollected EmbeddingBatchState = "collected"
	// EmbeddingBatchFailed is a job that failed or was cancelled. Its cards are
	// pending again.
	EmbeddingBatchFailed EmbeddingBatchState = "failed"
)

// EmbeddingBatch is one batch job submitted to the embedding provider. JobName
// is the provider's name for it, which collection uses to ask for the result.
type EmbeddingBatch struct {
	crud.BaseEntity
	JobName     string              `gorm:"not null;uniqueIndex"`
	Model       string              `gorm:"not null"`
	Source      string              `gorm:"not null"`
	State       EmbeddingBatchState `gorm:"not null"`
	CollectedAt *time.Time
}

// EmbeddingBatchCard links a card to the batch that embeds it.
type EmbeddingBatchCard struct {
	BatchID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CardID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (EmbeddingBatch) TableName() string { return "embedding_batches" }

func (EmbeddingBatchCard) TableName() string { return "embedding_batch_cards" }
