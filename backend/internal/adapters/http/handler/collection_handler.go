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

// CollectionHandler serves the authenticated Collections CRUD surface. Its
// routes are Secured; the router registers them behind SessionGuard.
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

func parseCollectionID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ungerr.BadRequestError("invalid collection id: " + raw)
	}
	return id, nil
}

type collectionBody struct {
	Title        string `json:"title" required:"true" minLength:"1" doc:"The Collection's title."`
	Description  string `json:"description,omitempty" doc:"Optional free-text description."`
	MaxCardCount *int   `json:"maxCardCount,omitempty" minimum:"1" doc:"Optional hard cap on the Collection's summed card quantity."`
}

type createCollectionInput struct {
	Body collectionBody
}

type listCollectionsInput struct{}

type collectionIDInput struct {
	ID string `path:"id" doc:"Collection ID"`
}

type updateCollectionInput struct {
	ID   string `path:"id" doc:"Collection ID"`
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

	summary, err := h.collectionSvc.Create(ctx, profileID, dto.CreateCollectionRequest{
		Title:        in.Body.Title,
		Description:  in.Body.Description,
		MaxCardCount: in.Body.MaxCardCount,
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return summary, nil
}

func (h *CollectionHandler) list(ctx context.Context, _ listCollectionsInput) ([]dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return nil, err
	}

	summaries, err := h.collectionSvc.List(ctx, profileID)
	if err != nil {
		return nil, err
	}

	return summaries, nil
}

func (h *CollectionHandler) get(ctx context.Context, in collectionIDInput) (dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	id, err := parseCollectionID(in.ID)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	summary, err := h.collectionSvc.Get(ctx, profileID, id)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return summary, nil
}

func (h *CollectionHandler) update(ctx context.Context, in updateCollectionInput) (dto.CollectionSummary, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	id, err := parseCollectionID(in.ID)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	if err := validateTitle(in.Body.Title); err != nil {
		return dto.CollectionSummary{}, err
	}

	summary, err := h.collectionSvc.Update(ctx, profileID, id, dto.UpdateCollectionRequest{
		Title:        in.Body.Title,
		Description:  in.Body.Description,
		MaxCardCount: in.Body.MaxCardCount,
	})
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return summary, nil
}

func (h *CollectionHandler) delete(ctx context.Context, in collectionIDInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	id, err := parseCollectionID(in.ID)
	if err != nil {
		return err
	}

	if err := h.collectionSvc.Delete(ctx, profileID, id); err != nil {
		return err
	}

	return nil
}

// Routes returns the Collections routes. Secured:true only sets OpenAPI
// security metadata; the router must pass SessionGuard to RegisterAll.
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
