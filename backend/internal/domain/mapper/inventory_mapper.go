package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

func ToInventoryEntry(e entity.InventoryEntry) dto.InventoryEntry {
	return dto.InventoryEntry{CardID: e.CardID, Quantity: e.Quantity}
}

// ToInventoryItem needs e.Card with its Rarity and ExpansionSet preloaded.
func ToInventoryItem(e entity.InventoryEntry) dto.InventoryItem {
	c := e.Card
	return dto.InventoryItem{
		Card: dto.CardSummary{
			ID:           c.ID,
			ExpansionSet: ToExpansionSetSummary(c.ExpansionSet),
			LocalID:      c.LocalID,
			Name:         c.Name,
			Category:     c.Category,
			Tags:         []string(c.Tags),
			Rarity:       dto.RaritySummary{ID: c.Rarity.ID, Code: c.Rarity.Code, Name: c.Rarity.Name},
			Illustrator:  c.Illustrator,
			ImageURL:     c.ImageURL,
		},
		Quantity: e.Quantity,
	}
}
