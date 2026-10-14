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

type createCardInput struct {
	Body struct {
		ExpansionSetID uuid.UUID `json:"expansionSetId" required:"true" doc:"The Expansion Set the Card belongs to (422 if unknown)."`
		LocalID        string    `json:"localId" required:"true" minLength:"1" maxLength:"32" doc:"The collector number within the set, e.g. \"001\". A number the set already has is a 409; it cannot be changed afterwards."`
		Name           string    `json:"name" required:"true" minLength:"1" maxLength:"200"`
		Category       string    `json:"category" required:"true" enum:"Pokémon,Trainer,Energi"`
		Tags           []string  `json:"tags,omitempty" enum:"Item,Supporter,Stadium" uniqueItems:"true" maxItems:"3" doc:"Only a Trainer card carries tags; Pokémon and Energi cards leave them empty (422 otherwise)."`
		RarityID       uuid.UUID `json:"rarityId,omitempty" doc:"A Rarity of the set's Game. Defaults to \"Tanpa Tanda\"."`
		Illustrator    string    `json:"illustrator,omitempty" maxLength:"200"`
	}
}

// updateCardInput carries every editable field of the Card, like a PUT does:
// what it leaves out is cleared, not kept. The set and collector number are the
// Card's identity and are not part of it.
type updateCardInput struct {
	ID   uuid.UUID `path:"id" doc:"Card ID"`
	Body struct {
		Name        string    `json:"name" required:"true" minLength:"1" maxLength:"200"`
		Category    string    `json:"category" required:"true" enum:"Pokémon,Trainer,Energi"`
		Tags        []string  `json:"tags,omitempty" enum:"Item,Supporter,Stadium" uniqueItems:"true" maxItems:"3" doc:"Only a Trainer card carries tags; Pokémon and Energi cards leave them empty (422 otherwise)."`
		RarityID    uuid.UUID `json:"rarityId" required:"true" doc:"A Rarity of the Card's Game (422 otherwise)."`
		Illustrator string    `json:"illustrator,omitempty" maxLength:"200" doc:"Left out, it clears the Card's illustrator."`
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

// cardLookupInput names the Card by its set and collector number, the identity
// the Admin enters it by, so a Card whose upload failed (and so is in no
// catalog list) can still be found.
type cardLookupInput struct {
	ExpansionSetID uuid.UUID `query:"expansionSetId" required:"true" doc:"The Expansion Set the Card belongs to."`
	LocalID        string    `query:"localId" required:"true" minLength:"1" maxLength:"32" doc:"The Card's collector number within the set."`
}

type cardDeleteInput struct {
	ID uuid.UUID `path:"id" doc:"Card ID"`
}

func (h *AdminCardsHandler) getCard(ctx context.Context, in cardLookupInput) (dto.AdminCard, error) {
	return h.cards.GetCard(ctx, dto.CardLookup{ExpansionSetID: in.ExpansionSetID, LocalID: in.LocalID})
}

func (h *AdminCardsHandler) deleteCard(ctx context.Context, in cardDeleteInput) error {
	return h.cards.DeleteCard(ctx, dto.CardDelete{ID: in.ID})
}

func (h *AdminCardsHandler) createCard(ctx context.Context, in createCardInput) (dto.AdminCard, error) {
	b := in.Body
	return h.cards.CreateCard(ctx, dto.AdminCardCreate{
		ExpansionSetID: b.ExpansionSetID, LocalID: b.LocalID, Name: b.Name, Category: b.Category,
		Tags: b.Tags, RarityID: b.RarityID, Illustrator: b.Illustrator,
	})
}

func (h *AdminCardsHandler) updateCard(ctx context.Context, in updateCardInput) (dto.AdminCard, error) {
	b := in.Body
	return h.cards.UpdateCard(ctx, dto.AdminCardUpdate{
		ID: in.ID, Name: b.Name, Category: b.Category, Tags: b.Tags, RarityID: b.RarityID, Illustrator: b.Illustrator,
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
		endpoint.New(endpoint.Endpoint[cardLookupInput, dto.AdminCard]{
			OperationID: "get-admin-card",
			Method:      http.MethodGet,
			Path:        "/admin/cards",
			Summary:     "Find the Card at an Expansion Set and collector number, including one with no image yet (404 if there is none)",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.getCard,
		}),
		endpoint.New(endpoint.Endpoint[createCardInput, dto.AdminCard]{
			OperationID: "create-admin-card",
			Method:      http.MethodPost,
			Path:        "/admin/cards",
			Summary:     "Create a Card, Curated; one already at the Expansion Set and collector number is a 409",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusCreated,
			Secured:     true,
			HandlerFunc: h.createCard,
		}),
		endpoint.New(endpoint.Endpoint[updateCardInput, dto.AdminCard]{
			OperationID: "update-admin-card",
			Method:      http.MethodPut,
			Path:        "/admin/cards/{id}",
			Summary:     "Replace the Card's name, category, tags, rarity and illustrator with the request's, keeping its set, number and image; the Card becomes Curated",
			Tags:        []string{"admin"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.updateCard,
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
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[cardDeleteInput]{
			OperationID: "delete-admin-card",
			Method:      http.MethodDelete,
			Path:        "/admin/cards/{id}",
			Summary:     "Delete the Card with its hosted image and embeddings; refused with 409 while a Collection holds it",
			Tags:        []string{"admin"},
			Secured:     true,
			HandlerFunc: h.deleteCard,
		}),
	}
}
