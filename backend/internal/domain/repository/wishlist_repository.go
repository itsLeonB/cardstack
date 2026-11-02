package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/gorm"
)

// WishlistStatus narrows a Wishlist listing by its derived status
// (GLOSSARY.md: Wishlist). Owned quantity is a Master Inventory sum, so the
// status is computed on read and never stored.
type WishlistStatus string

const (
	WishlistStatusAll       WishlistStatus = ""
	WishlistStatusNeeded    WishlistStatus = "needed"
	WishlistStatusFulfilled WishlistStatus = "fulfilled"
)

// WishlistFilter is one Wishlist listing query: the owner, the shared Card
// filters, and the status.
type WishlistFilter struct {
	ProfileID uuid.UUID
	Status    WishlistStatus
	Cards     CardFilter
}

// WishlistRepository is crud.Repository[entity.WishlistItem] plus the queries
// go-crud has no equivalent for: the Card listing with the owner's Master
// Inventory quantity (an aggregate and a join), the facets over that listing,
// the conditional delete behind Clear fulfilled, and the preset query for Cards
// the user owns none of.
type WishlistRepository interface {
	crud.Repository[entity.WishlistItem]
	// List pages the profile's Wishlist Items matching filter, each with its
	// Card, the wanted quantity and the owned quantity from Master Inventory.
	// Cards of unpublished Expansion Sets never appear.
	List(ctx context.Context, filter WishlistFilter) ([]CardResult, int64, error)
	// ListFacets returns each filter's options computed only from the Items
	// List would show.
	ListFacets(ctx context.Context, filter WishlistFilter) (CardFacets, error)
	// DeleteFulfilled removes every fulfilled item of the profile and returns
	// how many went.
	DeleteFulfilled(ctx context.Context, profileID uuid.UUID) (int, error)
	// FindOwnedNoneCards returns published Cards of the chosen Expansion Sets,
	// optionally narrowed by rarity, category and tag, of which the user owns
	// none in Master Inventory. Cards already on the Wishlist are excluded.
	FindOwnedNoneCards(ctx context.Context, profileID uuid.UUID, expansionSetIDs []uuid.UUID, filter CardFilter) ([]CardResult, error)
}

type wishlistRepository struct {
	crud.Repository[entity.WishlistItem]
}

func NewWishlistRepository(base crud.Repository[entity.WishlistItem]) WishlistRepository {
	return &wishlistRepository{Repository: base}
}

const wishlistColumns = cardResultColumns + `,
	wishlist_items.wanted_quantity AS wanted_quantity,
	COALESCE(master_inventory.quantity, 0) AS quantity`

// wishlistBase is the profile's Wishlist Items on their Cards, with the owned
// quantity alongside and the status applied.
func wishlistBase(db *gorm.DB, filter WishlistFilter) *gorm.DB {
	q := cardsWithoutAdminDrafts(db).
		Joins("JOIN wishlist_items ON wishlist_items.card_id = cards.id AND wishlist_items.profile_id = ?", filter.ProfileID).
		Joins("LEFT JOIN "+masterInventorySubquery+" AS master_inventory ON master_inventory.card_id = cards.id", filter.ProfileID)
	switch filter.Status {
	case WishlistStatusNeeded:
		q = q.Where("COALESCE(master_inventory.quantity, 0) < wishlist_items.wanted_quantity")
	case WishlistStatusFulfilled:
		q = q.Where("COALESCE(master_inventory.quantity, 0) >= wishlist_items.wanted_quantity")
	}

	return visibleCards(q, false)
}

func (r *wishlistRepository) List(ctx context.Context, filter WishlistFilter) ([]CardResult, int64, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := applyCardFilters(withCardResultJoins(wishlistBase(db, filter)), filter.Cards)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, ungerr.Wrap(err, "counting wishlist items")
	}

	var results []CardResult
	err = query.Session(&gorm.Session{}).
		Select(wishlistColumns).
		Order("expansion_sets.release_date DESC NULLS LAST, expansion_sets.id ASC, cards.local_id ASC, cards.name ASC, cards.id ASC").
		Limit(filter.Cards.Limit).
		Offset(filter.Cards.Offset).
		Find(&results).
		Error
	if err != nil {
		return nil, 0, ungerr.Wrap(err, "listing wishlist items")
	}

	return results, total, nil
}

func (r *wishlistRepository) ListFacets(ctx context.Context, filter WishlistFilter) (CardFacets, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return CardFacets{}, err
	}

	base := func(cards CardFilter) *gorm.DB {
		return wishlistBase(db, WishlistFilter{ProfileID: filter.ProfileID, Status: filter.Status, Cards: cards})
	}

	return listCardFacets(db, filter.Cards, base)
}

// DeleteFulfilled compares owned against wanted in the same statement, so no
// row can change status between the read and the delete.
func (r *wishlistRepository) DeleteFulfilled(ctx context.Context, profileID uuid.UUID) (int, error) {
	if profileID == uuid.Nil {
		return 0, nil
	}

	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return 0, err
	}

	result := db.Exec(`DELETE FROM wishlist_items w
		USING (
			SELECT wi.id
			FROM wishlist_items wi
			LEFT JOIN `+masterInventorySubquery+` AS master_inventory ON master_inventory.card_id = wi.card_id
			WHERE wi.profile_id = ?
			AND COALESCE(master_inventory.quantity, 0) >= wi.wanted_quantity
		) AS fulfilled
		WHERE w.id = fulfilled.id`, profileID, profileID)
	if result.Error != nil {
		return 0, ungerr.Wrap(result.Error, "clearing fulfilled wishlist items")
	}

	return int(result.RowsAffected), nil
}

// FindOwnedNoneCards returns Cards the user owns none of in Master Inventory,
// excluding Cards already on the Wishlist. Expansion Sets are filtered to the
// chosen ones, and the shared Card filters (rarity, category, tag) are applied.
// Only published Expansion Sets and their Cards are returned.
func (r *wishlistRepository) FindOwnedNoneCards(ctx context.Context, profileID uuid.UUID, expansionSetIDs []uuid.UUID, filter CardFilter) ([]CardResult, error) {
	db, err := r.GetGormInstance(ctx)
	if err != nil {
		return nil, err
	}

	// Cards the user owns none of: left join Master Inventory, keep only null
	query := cardsWithoutAdminDrafts(db).
		Joins("LEFT JOIN "+masterInventorySubquery+" AS master_inventory ON master_inventory.card_id = cards.id", profileID).
		Where("master_inventory.quantity IS NULL").
		// Exclude Cards already on the Wishlist
		Joins("LEFT JOIN wishlist_items ON wishlist_items.card_id = cards.id AND wishlist_items.profile_id = ?", profileID).
		Where("wishlist_items.id IS NULL")

	// Filter to chosen Expansion Sets
	if len(expansionSetIDs) > 0 {
		query = query.Where("cards.expansion_set_id IN ?", expansionSetIDs)
	}

	// Apply shared Card filters
	query = applyCardFilters(withCardResultJoins(query), filter)

	var results []CardResult
	err = query.Session(&gorm.Session{}).
		Select(cardResultColumns).
		Order("expansion_sets.release_date DESC NULLS LAST, expansion_sets.id ASC, cards.local_id ASC, cards.name ASC, cards.id ASC").
		Find(&results).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "finding owned-none cards for preset")
	}

	return results, nil
}
