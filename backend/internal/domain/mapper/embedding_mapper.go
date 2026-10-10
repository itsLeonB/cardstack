package mapper

import (
	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/pgvector/pgvector-go"
)

// ToCardEmbeddings converts the vectors of a succeeded embedding batch into card
// embedding rows for model and source. A key is kept only when it names a card
// the batch was submitted with. A key the batch never held, such as a card
// deleted since the submit, would break the card foreign key and fail the whole
// upsert, so those keys come back in unmatched instead.
func ToCardEmbeddings(vectors map[string][]float32, batchCards []uuid.UUID, model, source string) (rows []entity.CardEmbedding, unmatched []string) {
	known := make(map[uuid.UUID]bool, len(batchCards))
	for _, id := range batchCards {
		known[id] = true
	}

	for key, vector := range vectors {
		id, err := uuid.Parse(key)
		if err != nil || !known[id] {
			unmatched = append(unmatched, key)
			continue
		}
		rows = append(rows, entity.CardEmbedding{
			CardID:    id,
			Model:     model,
			Source:    source,
			Embedding: pgvector.NewVector(vector),
		})
	}
	return rows, unmatched
}
