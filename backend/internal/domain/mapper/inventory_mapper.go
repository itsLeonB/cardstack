package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
)

func ToInventoryEntry(e entity.InventoryEntry) dto.InventoryEntry {
	return dto.InventoryEntry{CardID: e.CardID, Quantity: e.Quantity}
}

func ToInventoryItem(r repository.CardResult) dto.InventoryItem {
	return dto.InventoryItem{Card: ToCardSummary(r), Quantity: r.Quantity}
}
