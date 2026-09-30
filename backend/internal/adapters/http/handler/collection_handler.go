package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/domain/collection"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	authkit "github.com/itsLeonB/go-authkit"
	"github.com/itsLeonB/ungerr"
)

// CollectionHandler serves the authenticated Collections CRUD surface.
type CollectionHandler struct {
	collectionSvc collection.CollectionService
	kit           *authkit.AuthKit
	transport     *authpkg.Transport
	profiles      authpkg.ProfileLookup
}

func NewCollectionHandler(collectionSvc collection.CollectionService, kit *authkit.AuthKit, profiles authpkg.ProfileLookup) *CollectionHandler {
	return &CollectionHandler{
		collectionSvc: collectionSvc,
		kit:           kit,
		transport:     authpkg.NewTransport(config.Global.Auth),
		profiles:      profiles,
	}
}

// requireUserID returns the user ID SessionGuard stashed in ctx; a failure
// here is unreachable behind SessionGuard.
func requireUserID(ctx context.Context) (uuid.UUID, error) {
	raw, _ := authpkg.UserID(ctx)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ungerr.UnauthorizedError("missing session")
	}

	return id, nil
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
	userID, err := requireUserID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	if err := validateTitle(in.Body.Title); err != nil {
		return dto.CollectionSummary{}, err
	}

	summary, err := h.collectionSvc.Create(ctx, userID, dto.CreateCollectionRequest{
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
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}

	summaries, err := h.collectionSvc.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	return summaries, nil
}

func (h *CollectionHandler) get(ctx context.Context, in collectionIDInput) (dto.CollectionSummary, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	id, err := parseCollectionID(in.ID)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	summary, err := h.collectionSvc.Get(ctx, userID, id)
	if err != nil {
		return dto.CollectionSummary{}, err
	}

	return summary, nil
}

func (h *CollectionHandler) update(ctx context.Context, in updateCollectionInput) (dto.CollectionSummary, error) {
	userID, err := requireUserID(ctx)
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

	summary, err := h.collectionSvc.Update(ctx, userID, id, dto.UpdateCollectionRequest{
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
	userID, err := requireUserID(ctx)
	if err != nil {
		return err
	}

	id, err := parseCollectionID(in.ID)
	if err != nil {
		return err
	}

	if err := h.collectionSvc.Delete(ctx, userID, id); err != nil {
		return err
	}

	return nil
}

// Routes builds its own SessionGuard per route: Secured:true only sets
// OpenAPI security metadata and does not enforce anything.
func (h *CollectionHandler) Routes() []endpoint.Registrable {
	sessionGuard := func(api huma.API) func(huma.Context, func(huma.Context)) {
		return authpkg.SessionGuard(api, h.kit, h.transport, h.profiles)
	}

	return []endpoint.Registrable{
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			endpoint.Register(api, endpoint.Endpoint[createCollectionInput, dto.CollectionSummary]{
				OperationID: "create-collection",
				Method:      http.MethodPost,
				Path:        "/collections",
				Summary:     "Create a Collection",
				Tags:        []string{"collections"},
				SuccessCode: http.StatusCreated,
				Secured:     true,
				HandlerFunc: h.create,
			}, withGuards(mw, sessionGuard(api))...)
		}),
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			endpoint.RegisterList(api, endpoint.ListEndpoint[listCollectionsInput, dto.CollectionSummary]{
				OperationID: "list-collections",
				Method:      http.MethodGet,
				Path:        "/collections",
				Summary:     "List the current user's own Collections",
				Tags:        []string{"collections"},
				Secured:     true,
				HandlerFunc: h.list,
			}, withGuards(mw, sessionGuard(api))...)
		}),
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			endpoint.Register(api, endpoint.Endpoint[collectionIDInput, dto.CollectionSummary]{
				OperationID: "get-collection",
				Method:      http.MethodGet,
				Path:        "/collections/{id}",
				Summary:     "Get one of the current user's own Collections",
				Tags:        []string{"collections"},
				SuccessCode: http.StatusOK,
				Secured:     true,
				HandlerFunc: h.get,
			}, withGuards(mw, sessionGuard(api))...)
		}),
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			endpoint.Register(api, endpoint.Endpoint[updateCollectionInput, dto.CollectionSummary]{
				OperationID: "update-collection",
				Method:      http.MethodPut,
				Path:        "/collections/{id}",
				Summary:     "Edit one of the current user's own Collections",
				Tags:        []string{"collections"},
				SuccessCode: http.StatusOK,
				Secured:     true,
				HandlerFunc: h.update,
			}, withGuards(mw, sessionGuard(api))...)
		}),
		registrableFunc(func(api huma.API, mw ...func(huma.Context, func(huma.Context))) {
			endpoint.RegisterNoBody(api, endpoint.NoBodyEndpoint[collectionIDInput]{
				OperationID: "delete-collection",
				Method:      http.MethodDelete,
				Path:        "/collections/{id}",
				Summary:     "Delete one of the current user's own Collections (hard delete, no undo)",
				Tags:        []string{"collections"},
				Secured:     true,
				HandlerFunc: h.delete,
			}, withGuards(mw, sessionGuard(api))...)
		}),
	}
}
