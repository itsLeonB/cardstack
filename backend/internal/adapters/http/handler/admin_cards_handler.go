package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

// AdminCardsHandler holds the admin routes that write Cards and their images
// (docs/adr/0019). Like AdminHandler's, they are mounted behind
// authpkg.AdminOnly.
type AdminCardsHandler struct {
	cards service.CardAdminService
}

func NewAdminCardsHandler(cards service.CardAdminService) *AdminCardsHandler {
	return &AdminCardsHandler{cards: cards}
}

type upsertCardInput struct {
	Body struct {
		ExpansionSetID uuid.UUID `json:"expansionSetId" required:"true" doc:"The Expansion Set the Card belongs to (422 if unknown)."`
		LocalID        string    `json:"localId" required:"true" minLength:"1" maxLength:"32" doc:"The collector number within the set, e.g. \"001\". Entering an existing set and number updates that Card."`
		Name           string    `json:"name" required:"true" minLength:"1" maxLength:"200"`
		Category       string    `json:"category" required:"true" enum:"Pokémon,Trainer,Energi"`
		Tag            string    `json:"tag,omitempty" enum:"Item,Supporter,Stadium" doc:"Only a Trainer card carries a tag; Pokémon and Energi cards leave it empty (422 otherwise)."`
		RarityID       uuid.UUID `json:"rarityId,omitempty" doc:"A Rarity of the set's Game. Defaults to \"Tanpa Tanda\"."`
		Illustrator    string    `json:"illustrator,omitempty" maxLength:"200"`
	}
}

type cardImageUploadInput struct {
	ID   uuid.UUID `path:"id" doc:"Card ID"`
	Body struct {
		ContentType string `json:"contentType" required:"true" enum:"image/jpeg,image/png,image/webp" doc:"The Content-Type the PUT must carry; it is part of the signature."`
	}
}

type cardImageConfirmInput struct {
	ID uuid.UUID `path:"id" doc:"Card ID"`
}

func (h *AdminCardsHandler) upsertCard(ctx context.Context, in upsertCardInput) (dto.AdminCard, error) {
	b := in.Body
	return h.cards.UpsertCard(ctx, dto.AdminCardUpsert{
		ExpansionSetID: b.ExpansionSetID, LocalID: b.LocalID, Name: b.Name, Category: b.Category,
		Tag: b.Tag, RarityID: b.RarityID, Illustrator: b.Illustrator,
	})
}

func (h *AdminCardsHandler) requestImageUpload(ctx context.Context, in cardImageUploadInput) (dto.CardImageUpload, error) {
	return h.cards.RequestImageUpload(ctx, dto.CardImageUploadRequest{CardID: in.ID, ContentType: in.Body.ContentType})
}

func (h *AdminCardsHandler) confirmImage(ctx context.Context, in cardImageConfirmInput) (dto.AdminCard, error) {
	return h.cards.ConfirmImage(ctx, dto.CardImageConfirm{CardID: in.ID})
}

// Routes returns every route AdminCardsHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *AdminCardsHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[upsertCardInput, dto.AdminCard]{
			OperationID: "upsert-admin-card",
			Method:      http.MethodPut,
			Path:        "/admin/cards",
			Summary:     "Create a Card, or update the one at the same Expansion Set and collector number; the Card becomes Curated",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.upsertCard,
		}),
		endpoint.New(endpoint.Endpoint[cardImageUploadInput, dto.CardImageUpload]{
			OperationID: "request-admin-card-image-upload",
			Method:      http.MethodPost,
			Path:        "/admin/cards/{id}/image-upload",
			Summary:     "Sign a short-lived PUT address for the Card's image; the server picks the object key",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.requestImageUpload,
		}),
		endpoint.New(endpoint.Endpoint[cardImageConfirmInput, dto.AdminCard]{
			OperationID: "confirm-admin-card-image",
			Method:      http.MethodPost,
			Path:        "/admin/cards/{id}/image-confirm",
			Summary:     "Check the uploaded image's first bytes and, if it is a JPEG, PNG or WebP, set it as the Card's image; otherwise delete it (422)",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.confirmImage,
		}),
	}
}
