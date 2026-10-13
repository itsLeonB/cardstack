package service

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/mapper"
	crud "github.com/itsLeonB/go-crud"
	"github.com/itsLeonB/ungerr"
	"gorm.io/datatypes"
)

const (
	// DefaultRarityName is the rarity of a Card entered without one: the promo
	// mark of the Indonesian SM-P set (docs/adr/0019).
	DefaultRarityName = "Tanpa Tanda"

	// cardImageUploadTTL is how long a signed upload address stays valid.
	cardImageUploadTTL = 5 * time.Minute

	// signatureBytes is the longest file header isImageSignature reads (WebP's
	// RIFF....WEBP).
	signatureBytes = 12

	categoryTrainer = "Trainer"
)

const (
	adminSetNotFoundMsg   = "expansion set not found"
	adminRarityInvalidMsg = "rarity not found in this expansion set's game"
	adminTagNotAllowedMsg = "only a Trainer card carries a tag"
	adminCardNotFoundMsg  = "card not found"
	noUploadMsg           = "no image has been uploaded for this card"
	notAnImageMsg         = "the uploaded file is not a JPEG, PNG or WebP image and was deleted"
	cardCollectedMsg      = "card is in a collection and cannot be deleted"
)

// CardAdminService is the Admin's write access to Cards and their images
// (docs/adr/0019). Every write marks the Card Curated.
type CardAdminService interface {
	// UpsertCard creates the Card at (Expansion Set, collector number) or
	// updates the one already there. A new Card has no image and stays out of
	// the catalog until ConfirmImage sets one.
	UpsertCard(ctx context.Context, req dto.AdminCardUpsert) (dto.AdminCard, error)
	// RequestImageUpload signs a short-lived PUT address for the Card's image
	// key, which the server derives from the Card's id.
	RequestImageUpload(ctx context.Context, req dto.CardImageUploadRequest) (dto.CardImageUpload, error)
	// ConfirmImage checks the first bytes of the object uploaded for the Card
	// and, only if they are a JPEG, PNG or WebP signature, sets the Card's
	// image key. Otherwise it deletes the object and refuses.
	ConfirmImage(ctx context.Context, req dto.CardImageConfirm) (dto.AdminCard, error)
	// DeleteCard removes the Card, its hosted image object and (by cascade) its
	// embeddings and batch membership. A Card some Collection still holds is
	// refused with a conflict, because deleting it would erase a collector's
	// records.
	DeleteCard(ctx context.Context, req dto.CardDelete) error
}

type cardAdminService struct {
	transactor crud.Transactor
	cards      crud.Repository[entity.Card]
	sets       crud.Repository[entity.ExpansionSet]
	rarities   crud.Repository[entity.Rarity]
	entries    crud.Repository[entity.InventoryEntry]
	store      objectstore.ObjectStore
	images     mapper.ImageHost
}

func NewCardAdminService(
	transactor crud.Transactor,
	cards crud.Repository[entity.Card],
	sets crud.Repository[entity.ExpansionSet],
	rarities crud.Repository[entity.Rarity],
	entries crud.Repository[entity.InventoryEntry],
	store objectstore.ObjectStore,
	images mapper.ImageHost,
) CardAdminService {
	return &cardAdminService{transactor: transactor, cards: cards, sets: sets, rarities: rarities, entries: entries, store: store, images: images}
}

func (s *cardAdminService) UpsertCard(ctx context.Context, req dto.AdminCardUpsert) (dto.AdminCard, error) {
	if req.Tag != "" && req.Category != categoryTrainer {
		return dto.AdminCard{}, ungerr.UnprocessableEntityError(adminTagNotAllowedMsg)
	}

	var saved entity.Card
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		// Locking the set serializes writers to its cards, so two saves of the
		// same number cannot both insert.
		set, err := s.getSet(ctx, req.ExpansionSetID, true)
		if err != nil {
			return err
		}
		tags := datatypes.JSONSlice[string]{}
		if req.Tag != "" {
			tags = append(tags, req.Tag)
		}

		card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
			Model: entity.Card{ExpansionSetID: set.ID, LocalID: req.LocalID},
		})
		if err != nil {
			return err
		}
		if card.IsZero() {
			rarityID, err := s.resolveRarity(ctx, set, req.RarityID)
			if err != nil {
				return err
			}
			saved, err = s.cards.Insert(ctx, entity.Card{
				ExpansionSetID: set.ID, LocalID: req.LocalID, Name: req.Name, Category: req.Category,
				Illustrator: req.Illustrator, Tags: tags, RarityID: rarityID, Attributes: datatypes.JSONMap{}, Curated: true,
			})
			return err
		}

		// An edit that leaves the rarity or the illustrator out keeps what the
		// Card has, so fixing a name does not erase what an import filled in.
		if req.RarityID != uuid.Nil {
			if card.RarityID, err = s.resolveRarity(ctx, set, req.RarityID); err != nil {
				return err
			}
		}
		if req.Illustrator != "" {
			card.Illustrator = req.Illustrator
		}
		card.Name, card.Category, card.Tags, card.Curated = req.Name, req.Category, tags, true
		saved, err = s.cards.Update(ctx, card)
		return err
	})
	if err != nil {
		return dto.AdminCard{}, err
	}
	return mapper.ToAdminCard(s.images, saved), nil
}

func (s *cardAdminService) getSet(ctx context.Context, id uuid.UUID, forUpdate bool) (entity.ExpansionSet, error) {
	if id == uuid.Nil {
		return entity.ExpansionSet{}, ungerr.UnprocessableEntityError(adminSetNotFoundMsg)
	}
	set, err := s.sets.FindFirst(ctx, crud.Specification[entity.ExpansionSet]{
		Model:     entity.ExpansionSet{BaseEntity: crud.BaseEntity{ID: id}},
		ForUpdate: forUpdate,
	})
	if err != nil {
		return entity.ExpansionSet{}, err
	}
	if set.IsZero() {
		return entity.ExpansionSet{}, ungerr.UnprocessableEntityError(adminSetNotFoundMsg)
	}
	return set, nil
}

// resolveRarity returns the chosen Rarity of set's Game, or the Game's
// DefaultRarityName one when none is chosen.
func (s *cardAdminService) resolveRarity(ctx context.Context, set entity.ExpansionSet, id uuid.UUID) (uuid.UUID, error) {
	model := entity.Rarity{GameID: set.GameID, Name: DefaultRarityName}
	if id != uuid.Nil {
		model = entity.Rarity{BaseEntity: crud.BaseEntity{ID: id}, GameID: set.GameID}
	}
	rarity, err := s.rarities.FindFirst(ctx, crud.Specification[entity.Rarity]{Model: model})
	if err != nil {
		return uuid.Nil, err
	}
	if rarity.IsZero() {
		return uuid.Nil, ungerr.UnprocessableEntityError(adminRarityInvalidMsg)
	}
	return rarity.ID, nil
}

func (s *cardAdminService) getCard(ctx context.Context, id uuid.UUID, forUpdate bool) (entity.Card, error) {
	if id == uuid.Nil {
		return entity.Card{}, ungerr.NotFoundError(adminCardNotFoundMsg)
	}
	card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
		Model:     entity.Card{BaseEntity: crud.BaseEntity{ID: id}},
		ForUpdate: forUpdate,
	})
	if err != nil {
		return entity.Card{}, err
	}
	if card.IsZero() {
		return entity.Card{}, ungerr.NotFoundError(adminCardNotFoundMsg)
	}
	return card, nil
}

// cardImageKey is the object key of a Card's hosted image, the same one the
// ingester uses (docs/adr/0016).
func cardImageKey(id uuid.UUID) string { return "cards/" + id.String() }

func (s *cardAdminService) RequestImageUpload(ctx context.Context, req dto.CardImageUploadRequest) (dto.CardImageUpload, error) {
	card, err := s.getCard(ctx, req.CardID, false)
	if err != nil {
		return dto.CardImageUpload{}, err
	}
	url, expiresAt, err := s.store.PresignPut(ctx, cardImageKey(card.ID), req.ContentType, cardImageUploadTTL)
	if err != nil {
		return dto.CardImageUpload{}, err
	}
	return dto.CardImageUpload{URL: url, ContentType: req.ContentType, ExpiresAt: expiresAt}, nil
}

func (s *cardAdminService) ConfirmImage(ctx context.Context, req dto.CardImageConfirm) (dto.AdminCard, error) {
	// An unlocked look first, so an unknown Card is a 404 before any bucket call.
	if _, err := s.getCard(ctx, req.CardID, false); err != nil {
		return dto.AdminCard{}, err
	}
	key := cardImageKey(req.CardID)

	// Bucket calls stay outside the transaction, so a slow one never holds the
	// Card's row lock.
	head, err := s.store.ReadHead(ctx, key, signatureBytes)
	if errors.Is(err, objectstore.ErrNotFound) {
		return dto.AdminCard{}, ungerr.UnprocessableEntityError(noUploadMsg)
	}
	if err != nil {
		return dto.AdminCard{}, err
	}

	if !isImageSignature(head) {
		if err := s.store.Delete(ctx, key); err != nil {
			logger.Errorf("deleting rejected upload %q: %v", key, err)
		}
		// A re-upload has already replaced the old image under the same key, so
		// a Card that pointed at it would now point at nothing.
		if _, err := s.updateCard(ctx, req.CardID, func(card *entity.Card) { card.ImageKey = "" }); err != nil {
			return dto.AdminCard{}, err
		}
		return dto.AdminCard{}, ungerr.UnprocessableEntityError(notAnImageMsg)
	}

	saved, err := s.updateCard(ctx, req.CardID, func(card *entity.Card) {
		// Curated here too: an Admin may confirm an image for a Card that
		// ingestion wrote and no UpsertCard has marked yet.
		card.ImageKey, card.Curated = key, true
	})
	if err != nil {
		return dto.AdminCard{}, err
	}
	return mapper.ToAdminCard(s.images, saved), nil
}

// updateCard re-reads the Card under a row lock, applies change and saves it,
// so a concurrent UpsertCard's write is not overwritten with a stale row.
func (s *cardAdminService) updateCard(ctx context.Context, id uuid.UUID, change func(*entity.Card)) (entity.Card, error) {
	var saved entity.Card
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		card, err := s.getCard(ctx, id, true)
		if err != nil {
			return err
		}
		change(&card)
		saved, err = s.cards.Update(ctx, card)
		return err
	})
	return saved, err
}

func (s *cardAdminService) DeleteCard(ctx context.Context, req dto.CardDelete) error {
	// A zero field would drop out of the Model below and widen the match.
	if req.ExpansionSetID == uuid.Nil || req.LocalID == "" {
		return ungerr.NotFoundError(adminCardNotFoundMsg)
	}
	var deleted uuid.UUID
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		// The lock blocks a concurrent Inventory Entry insert (its foreign key
		// check shares the Card row), so the check below cannot go stale.
		card, err := s.cards.FindFirst(ctx, crud.Specification[entity.Card]{
			Model:     entity.Card{ExpansionSetID: req.ExpansionSetID, LocalID: req.LocalID},
			ForUpdate: true,
		})
		if err != nil {
			return err
		}
		if card.IsZero() {
			return ungerr.NotFoundError(adminCardNotFoundMsg)
		}
		held, err := s.entries.FindFirst(ctx, crud.Specification[entity.InventoryEntry]{
			Model: entity.InventoryEntry{CardID: card.ID},
		})
		if err != nil {
			return err
		}
		if !held.IsZero() {
			return ungerr.ConflictError(cardCollectedMsg)
		}
		deleted = card.ID
		return s.cards.Delete(ctx, card)
	})
	if err != nil {
		return err
	}

	// After the commit, so a failed delete never leaves a Card without its
	// image. The key comes from the id, not the row, because an upload that was
	// never confirmed leaves an object under it too. A failure leaves an orphan
	// object, which the logged key lets the owner remove.
	key := cardImageKey(deleted)
	if err := s.store.Delete(ctx, key); err != nil {
		logger.Errorf("deleting image object %q of a deleted card: %v", key, err)
	}
	return nil
}

var (
	jpegMagic = []byte{0xff, 0xd8, 0xff}
	pngMagic  = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
)

// isImageSignature reports whether head starts like a JPEG, a PNG or a WebP
// (a RIFF container whose form type is WEBP).
func isImageSignature(head []byte) bool {
	return bytes.HasPrefix(head, jpegMagic) ||
		bytes.HasPrefix(head, pngMagic) ||
		(len(head) >= signatureBytes && bytes.HasPrefix(head, []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")))
}
