package mapper

import (
	"testing"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

func TestToCardEmbeddings_KeepsOnlyTheBatchsCards(t *testing.T) {
	inBatch, deleted := uuid.New(), uuid.New()
	vector := []float32{1, 0}

	rows, unmatched := ToCardEmbeddings(
		map[string][]float32{
			inBatch.String(): vector,
			deleted.String(): vector,
			"not-a-uuid":     vector,
		},
		[]uuid.UUID{inBatch},
		"model-a",
		"catalog",
	)

	assert.Equal(t, []entity.CardEmbedding{{
		CardID:    inBatch,
		Model:     "model-a",
		Source:    "catalog",
		Embedding: pgvector.NewVector(vector),
	}}, rows)
	assert.ElementsMatch(t, []string{deleted.String(), "not-a-uuid"}, unmatched,
		"a key the batch never held would break the card foreign key, so it is reported instead")
}

func TestToCardEmbeddings_NoVectorsGivesNoRows(t *testing.T) {
	rows, unmatched := ToCardEmbeddings(nil, []uuid.UUID{uuid.New()}, "model-a", "catalog")

	assert.Empty(t, rows)
	assert.Empty(t, unmatched)
}
