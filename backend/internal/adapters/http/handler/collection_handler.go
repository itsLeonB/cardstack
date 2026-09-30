package handler

import (
	"context"
	"net/http"

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

// CollectionHandler serves the authenticated Collections CRUD surface
// (ticket 06): a user creates, lists, views, edits, and deletes their own
// Collections. Every route requires a session, so - unlike CatalogHandler -
// it builds its own SessionGuard the same way AuthHandler's logout/refresh/
// me routes do (see Routes()): Secured:true on endpoint.Endpoint only sets
// OpenAPI security metadata, it doesn't attach SessionGuard as an enforced
// middleware on its own.
type CollectionHandler struct {
	collectionSvc collection.CollectionService
	kit           *authkit.AuthKit
	transport     *authpkg.Transport
	profiles      authpkg.ProfileLookup
}

// NewCollectionHandler builds a CollectionHandler.
func NewCollectionHandler(collectionSvc collection.CollectionService, kit *authkit.AuthKit, profiles authpkg.ProfileLookup) *CollectionHandler {
	return &CollectionHandler{
		collectionSvc: collectionSvc,
		kit:           kit,
		transport:     authpkg.NewTransport(config.Global.Auth),
		profiles:      profiles,
	}
}

// requireUserID reads the authenticated user's ID out of ctx (stashed by
// SessionGuard) and parses it. Both failure branches shouldn't be reachable
// in practice once SessionGuard runs first - it never calls next with an
// unset/malformed userID - but are handled defensively rather than assumed.
func requireUserID(ctx context.Context) (uuid.UUID, error) {
	raw, ok := authpkg.UserID(ctx)
	if !ok || raw == "" {
		return uuid.Nil, ungerr.UnauthorizedError("missing session")
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ungerr.UnauthorizedError("invalid session")
	}

	return id, nil
}

// parseCollectionID validates a Collection ID path parameter, rejecting a
// malformed one as a 400 before any query runs.
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

// Routes returns every route CollectionHandler exposes, for registration
// via endpoint.RegisterAll. Every route needs SessionGuard actually
// enforced (not just declared via Secured:true), so each is wrapped in a
// registrableFunc that appends it at Register-call time - mirrors
// auth_handler.go's logout/refresh/me routes, the existing precedent for
// this (see registrableFunc/withGuards, defined in auth_handler.go).
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
