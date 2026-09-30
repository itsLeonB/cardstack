package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	"github.com/itsLeonB/ungerr"
)

type CollectionHandler struct {
	collectionSvc service.CollectionService
}

func NewCollectionHandler(collectionSvc service.CollectionService) *CollectionHandler {
	return &CollectionHandler{collectionSvc: collectionSvc}
}

// validateTitle rejects whitespace-only titles, which minLength:"1" lets through.
func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return ungerr.BadRequestError("title must not be blank")
	}
	return nil
}

type collectionBody struct {
	Title        string `json:"title" required:"true" minLength:"1" doc:"The Collection's title."`
	Description  string `json:"description,omitempty" doc:"Optional free-text description."`
	MaxCardCount int    `json:"maxCardCount,omitempty" minimum:"0" maximum:"2147483647" doc:"Hard cap on the Collection's summed card quantity; 0 means no limit."`
}

type createCollectionInput struct {
	Body collectionBody
}

type listCollectionsInput struct{}

type collectionIDInput struct {
	ID uuid.UUID `path:"id" doc:"Collection ID"`
}

type updateCollectionInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Body collectionBody
}

func (h *CollectionHandler) create(ctx context.Context, in createCollectionInput) (dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	if err := validateTitle(in.Body.Title); err != nil {
		return dto.CollectionSummary{}, err
	}

	return h.collectionSvc.Create(ctx, profileID, dto.CollectionRequest{
		Title:        in.Body.Title,
		Description:  in.Body.Description,
		MaxCardCount: in.Body.MaxCardCount,
	})
}

func (h *CollectionHandler) list(ctx context.Context, _ listCollectionsInput) ([]dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, err
	}

	return h.collectionSvc.List(ctx, profileID)
}

func (h *CollectionHandler) get(ctx context.Context, in collectionIDInput) (dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return h.collectionSvc.Get(ctx, profileID, in.ID)
}

func (h *CollectionHandler) update(ctx context.Context, in updateCollectionInput) (dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	if err := validateTitle(in.Body.Title); err != nil {
		return dto.CollectionSummary{}, err
	}

	return h.collectionSvc.Update(ctx, profileID, in.ID, dto.CollectionRequest{
		Title:        in.Body.Title,
		Description:  in.Body.Description,
		MaxCardCount: in.Body.MaxCardCount,
	})
}

func (h *CollectionHandler) delete(ctx context.Context, in collectionIDInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	return h.collectionSvc.Delete(ctx, profileID, in.ID)
}

// Routes sets Secured:true only sets OpenAPI security metadata; the router must pass
// SessionGuard to RegisterAll.
func (h *CollectionHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[createCollectionInput, dto.CollectionSummary]{
			OperationID: "create-collection",
			Method:      http.MethodPost,
			Path:        "/collections",
			Summary:     "Create a Collection",
			Tags:        []string{"collections"},
			SuccessCode: http.StatusCreated,
			Secured:     true,
			HandlerFunc: h.create,
		}),
		endpoint.NewList(endpoint.ListEndpoint[listCollectionsInput, dto.CollectionSummary]{
			OperationID: "list-collections",
			Method:      http.MethodGet,
			Path:        "/collections",
			Summary:     "List the current user's own Collections",
			Tags:        []string{"collections"},
			Secured:     true,
			HandlerFunc: h.list,
		}),
		endpoint.New(endpoint.Endpoint[collectionIDInput, dto.CollectionSummary]{
			OperationID: "get-collection",
			Method:      http.MethodGet,
			Path:        "/collections/{id}",
			Summary:     "Get one of the current user's own Collections",
			Tags:        []string{"collections"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.get,
		}),
		endpoint.New(endpoint.Endpoint[updateCollectionInput, dto.CollectionSummary]{
			OperationID: "update-collection",
			Method:      http.MethodPut,
			Path:        "/collections/{id}",
			Summary:     "Edit one of the current user's own Collections",
			Tags:        []string{"collections"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.update,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[collectionIDInput]{
			OperationID: "delete-collection",
			Method:      http.MethodDelete,
			Path:        "/collections/{id}",
			Summary:     "Delete one of the current user's own Collections (hard delete, no undo)",
			Tags:        []string{"collections"},
			Secured:     true,
			HandlerFunc: h.delete,
		}),
	}
}
