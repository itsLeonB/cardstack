package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/ungerr"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCollectionNotFound = errors.New("collection not found")
	ErrEntryExists        = errors.New("inventory entry already exists")
	ErrEntryNotFound      = errors.New("inventory entry not found")
)

// InventoryItemResult is a Card (joined with its Rarity and Expansion Set)
// plus its quantity in a Collection.
type InventoryItemResult struct {
	CardResult
	Quantity int
}

// InventoryRepository is the persistence access Inventory Entries need.
type InventoryRepository interface {
	// ListItems returns a Collection's Cards with quantities, in the same
	// order as catalog search.
	ListItems(ctx context.Context, collectionID uuid.UUID) ([]InventoryItemResult, error)
	// FindEntry returns the entry for the Card in the Collection, or a zero
	// entry (IsZero) when there is none.
	FindEntry(ctx context.Context, collectionID, cardID uuid.UUID) (entity.InventoryEntry, error)
	// SumQuantity returns the Collection's summed quantity (0 when empty).
	SumQuantity(ctx context.Context, collectionID uuid.UUID) (int, error)
	// CardExists reports whether a Card with that ID exists.
	CardExists(ctx context.Context, cardID uuid.UUID) (bool, error)
	// InsertEntry inserts a new entry; ErrEntryExists if the Card is already
	// in the Collection.
	InsertEntry(ctx context.Context, entry entity.InventoryEntry) error
	// UpdateQuantity sets an existing entry's quantity; ErrEntryNotFound if
	// there is no such entry.
	UpdateQuantity(ctx context.Context, collectionID, cardID uuid.UUID, quantity int) error
	// WithLockedCollection runs fn in a transaction holding a row lock on the
	// profile's Collection, so concurrent writes to it serialize. fn gets a
	// repository bound to that transaction and the locked Collection.
	// ErrCollectionNotFound if the profile has no such Collection.
	WithLockedCollection(ctx context.Context, profileID, collectionID uuid.UUID, fn func(tx InventoryRepository, c entity.Collection) error) error
	// DeleteEntry removes the Card's entry from the Collection.
	DeleteEntry(ctx context.Context, collectionID, cardID uuid.UUID) error
}

type inventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepository{db: db}
}

func (r *inventoryRepository) ListItems(ctx context.Context, collectionID uuid.UUID) ([]InventoryItemResult, error) {
	var items []InventoryItemResult
	err := r.db.WithContext(ctx).
		Table("inventory_entries").
		Joins("JOIN cards ON cards.id = inventory_entries.card_id").
		Joins("JOIN rarities ON rarities.id = cards.rarity_id").
		Joins("JOIN expansion_sets ON expansion_sets.id = cards.expansion_set_id").
		Where("inventory_entries.collection_id = ?", collectionID).
		Select(cardResultColumns + ", inventory_entries.quantity AS quantity").
		Order("expansion_sets.release_date ASC NULLS LAST, expansion_sets.id ASC, cards.local_id ASC, cards.name ASC, cards.id ASC").
		Find(&items).
		Error
	if err != nil {
		return nil, ungerr.Wrap(err, "listing inventory items")
	}

	return items, nil
}

func (r *inventoryRepository) FindEntry(ctx context.Context, collectionID, cardID uuid.UUID) (entity.InventoryEntry, error) {
	var entry entity.InventoryEntry
	err := r.db.WithContext(ctx).
		Where("collection_id = ? AND card_id = ?", collectionID, cardID).
		Limit(1).
		Find(&entry).
		Error
	if err != nil {
		return entity.InventoryEntry{}, ungerr.Wrap(err, "finding inventory entry")
	}

	return entry, nil
}

func (r *inventoryRepository) SumQuantity(ctx context.Context, collectionID uuid.UUID) (int, error) {
	var sum int
	err := r.db.WithContext(ctx).
		Model(&entity.InventoryEntry{}).
		Where("collection_id = ?", collectionID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&sum).
		Error
	if err != nil {
		return 0, ungerr.Wrap(err, "summing inventory quantity")
	}

	return sum, nil
}

func (r *inventoryRepository) CardExists(ctx context.Context, cardID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Card{}).Where("id = ?", cardID).Count(&count).Error
	if err != nil {
		return false, ungerr.Wrap(err, "checking card existence")
	}

	return count > 0, nil
}

func (r *inventoryRepository) InsertEntry(ctx context.Context, entry entity.InventoryEntry) error {
	err := r.db.WithContext(ctx).Create(&entry).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 = unique_violation
		return ErrEntryExists
	}
	if err != nil {
		return ungerr.Wrap(err, "inserting inventory entry")
	}

	return nil
}

func (r *inventoryRepository) UpdateQuantity(ctx context.Context, collectionID, cardID uuid.UUID, quantity int) error {
	res := r.db.WithContext(ctx).
		Model(&entity.InventoryEntry{}).
		Where("collection_id = ? AND card_id = ?", collectionID, cardID).
		Update("quantity", quantity)
	if res.Error != nil {
		return ungerr.Wrap(res.Error, "updating inventory entry")
	}
	if res.RowsAffected == 0 {
		return ErrEntryNotFound
	}

	return nil
}

func (r *inventoryRepository) WithLockedCollection(ctx context.Context, profileID, collectionID uuid.UUID, fn func(tx InventoryRepository, c entity.Collection) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c entity.Collection
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND profile_id = ?", collectionID, profileID).
			Limit(1).
			Find(&c).
			Error
		if err != nil {
			return ungerr.Wrap(err, "locking collection")
		}
		if c.IsZero() {
			return ErrCollectionNotFound
		}

		return fn(&inventoryRepository{db: tx}, c)
	})
}

func (r *inventoryRepository) DeleteEntry(ctx context.Context, collectionID, cardID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("collection_id = ? AND card_id = ?", collectionID, cardID).
		Delete(&entity.InventoryEntry{}).
		Error
	if err != nil {
		return ungerr.Wrap(err, "deleting inventory entry")
	}

	return nil
}
