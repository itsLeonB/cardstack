package mapper

import (
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
)

// ToBinderLayout converts a binder's Layout and Leaf count into its summary.
// Capacity is the Slot count: each Leaf has two Pages of Rows × Columns Slots.
func ToBinderLayout(layout entity.CollectionLayout, leafCount int) dto.BinderLayout {
	return dto.BinderLayout{
		Rows:      layout.Rows,
		Columns:   layout.Columns,
		LeafCount: leafCount,
		Capacity:  leafCount * 2 * layout.Rows * layout.Columns,
	}
}

// ToBinderSlots lays one Page's occupied Slots onto its grid in grid order.
// A position with no entry in slots is empty.
func ToBinderSlots(host ImageHost, layout entity.CollectionLayout, slots []entity.BinderSlot) []dto.BinderSlot {
	byPosition := make(map[[2]int]entity.BinderSlot, len(slots))
	for _, slot := range slots {
		byPosition[[2]int{slot.Row, slot.Column}] = slot
	}

	grid := make([]dto.BinderSlot, 0, layout.Rows*layout.Columns)
	for row := 1; row <= layout.Rows; row++ {
		for column := 1; column <= layout.Columns; column++ {
			out := dto.BinderSlot{Row: row, Column: column, State: dto.BinderSlotEmpty}
			if slot, ok := byPosition[[2]int{row, column}]; ok {
				if slot.CardID != nil {
					card := ToBinderCardSummary(host, slot.Card)
					out.State = dto.BinderSlotCard
					out.Card = &card
				} else {
					out.State = dto.BinderSlotUnidentified
				}
			}
			grid = append(grid, out)
		}
	}

	return grid
}

// ToBinderCardSummary converts a Card preloaded with its Rarity and Expansion
// Set into the same summary the match endpoint returns.
func ToBinderCardSummary(host ImageHost, card entity.Card) dto.CardSummary {
	return dto.CardSummary{
		ID:           card.ID,
		ExpansionSet: ToExpansionSetSummary(host, card.ExpansionSet),
		LocalID:      card.LocalID,
		Name:         card.Name,
		Category:     card.Category,
		Tags:         []string(card.Tags),
		Rarity:       ToRaritySummary(card.Rarity),
		Illustrator:  card.Illustrator,
		ImageURL:     host.URL(card.ImageKey),
	}
}
