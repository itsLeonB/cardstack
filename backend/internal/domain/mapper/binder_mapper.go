package mapper

import (
	"cmp"
	"slices"

	"github.com/google/uuid"
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

// ToBinderPage builds one Page of a binder from its occupied Slots. page is
// counted in turning order, and cardTotal and unidentified count the whole binder.
func ToBinderPage(host ImageHost, page int, layout entity.CollectionLayout, leafCount int, slots []entity.BinderSlot, cardTotal, unidentified int) dto.BinderPage {
	leaf, side := LeafOf(page)
	return dto.BinderPage{
		Page:              page,
		Leaf:              leaf,
		Side:              side,
		Layout:            ToBinderLayout(layout, leafCount),
		CardTotal:         cardTotal,
		UnidentifiedCount: unidentified,
		Slots:             ToBinderSlots(host, layout, slots),
	}
}

// PageOf is the number of the Page on one face of a Leaf. Pages are numbered in
// turning order, so Leaf n holds Pages 2n-1 (front) and 2n (back).
func PageOf(leaf int, side string) int {
	if side == dto.BinderSideBack {
		return 2 * leaf
	}
	return 2*leaf - 1
}

// LeafOf is the Leaf and face a Page belongs to, the inverse of PageOf.
func LeafOf(page int) (int, string) {
	if page%2 == 0 {
		return page / 2, dto.BinderSideBack
	}
	return (page + 1) / 2, dto.BinderSideFront
}

// ToCardPositions groups the Slots holding a Card by that Card, each Slot as its
// position, in Page order. An Unidentified Slot holds no Card, so it is skipped.
func ToCardPositions(slots []entity.BinderSlot) map[uuid.UUID][]dto.BinderSlotTarget {
	positions := make(map[uuid.UUID][]dto.BinderSlotTarget)
	for _, slot := range slots {
		if slot.CardID == nil {
			continue
		}
		page := PageOf(slot.Leaf.Position, slot.Side)
		positions[*slot.CardID] = append(positions[*slot.CardID], dto.BinderSlotTarget{Page: page, Row: slot.Row, Column: slot.Column})
	}
	for _, list := range positions {
		slices.SortFunc(list, func(a, b dto.BinderSlotTarget) int {
			return cmp.Or(cmp.Compare(a.Page, b.Page), cmp.Compare(a.Row, b.Row), cmp.Compare(a.Column, b.Column))
		})
	}

	return positions
}
