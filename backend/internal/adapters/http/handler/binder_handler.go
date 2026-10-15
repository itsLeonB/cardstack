package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

type BinderHandler struct {
	binderSvc service.BinderService
}

func NewBinderHandler(binderSvc service.BinderService) *BinderHandler {
	return &BinderHandler{binderSvc: binderSvc}
}

type BinderGridBody struct {
	Rows    int `json:"rows" required:"true" minimum:"1" maximum:"6" doc:"Rows of Slots on each Page, from 1 to 6."`
	Columns int `json:"columns" required:"true" minimum:"1" maximum:"6" doc:"Columns of Slots on each Page, from 1 to 6."`
}

type binderLayoutBody struct {
	BinderGridBody
	LeafCount int `json:"leafCount,omitempty" default:"10" minimum:"1" maximum:"100" doc:"Number of Leaves the binder starts with, from 1 to 100."`
}

type binderLeafBody struct {
	Leaf      int    `json:"leaf" required:"true" minimum:"1" maximum:"100" doc:"Number of the Leaf the new Leaf goes beside."`
	Placement string `json:"placement" required:"true" enum:"before,after" doc:"Whether the new Leaf goes before or after that Leaf."`
}

type binderCreateInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Body binderLayoutBody
}

type binderGridInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Body BinderGridBody
}

type binderLeafInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Body binderLeafBody
}

type binderLeafPathInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Leaf int       `path:"leaf" minimum:"1" doc:"Number of the Leaf to remove, counted from 1 in turning order."`
}

type binderPageInput struct {
	ID   uuid.UUID `path:"id" doc:"Collection ID"`
	Page int       `path:"page" minimum:"1" doc:"Page number in turning order, counted from 1."`
}

// binderSlotPathInput addresses one Slot of a binder by its Page, row and column.
type binderSlotPathInput struct {
	ID     uuid.UUID `path:"id" doc:"Collection ID"`
	Page   int       `path:"page" minimum:"1" doc:"Page number in turning order, counted from 1."`
	Row    int       `path:"row" minimum:"1" doc:"Row of the Slot on its Page, counted from 1."`
	Column int       `path:"column" minimum:"1" doc:"Column of the Slot on its Page, counted from 1."`
}

// binderSlotCardInput is binderSlotPathInput with a body naming the Card.
type binderSlotCardInput struct {
	ID     uuid.UUID `path:"id" doc:"Collection ID"`
	Page   int       `path:"page" minimum:"1" doc:"Page number in turning order, counted from 1."`
	Row    int       `path:"row" minimum:"1" doc:"Row of the Slot on its Page, counted from 1."`
	Column int       `path:"column" minimum:"1" doc:"Column of the Slot on its Page, counted from 1."`
	Body   binderSlotCardBody
}

type binderSlotCardBody struct {
	CardID uuid.UUID `json:"cardId" required:"true" doc:"The Card to place in the Slot. A Slot that held a Card gives it up; an Unidentified Slot stops being Unidentified."`
}

func (h *BinderHandler) create(ctx context.Context, in binderCreateInput) (dto.BinderLayout, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return h.binderSvc.CreateBinder(ctx, dto.BinderLayoutRequest{
		ProfileID:    profileID,
		CollectionID: in.ID,
		Rows:         in.Body.Rows,
		Columns:      in.Body.Columns,
		LeafCount:    in.Body.LeafCount,
	})
}

func (h *BinderHandler) changeGrid(ctx context.Context, in binderGridInput) (dto.BinderLayout, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return h.binderSvc.ChangeGrid(ctx, dto.BinderGridRequest{
		ProfileID:    profileID,
		CollectionID: in.ID,
		Rows:         in.Body.Rows,
		Columns:      in.Body.Columns,
	})
}

func (h *BinderHandler) remove(ctx context.Context, in collectionIDInput) error {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return err
	}

	return h.binderSvc.RemoveBinder(ctx, dto.CollectionLookup{ProfileID: profileID, ID: in.ID})
}

func (h *BinderHandler) addLeaf(ctx context.Context, in binderLeafInput) (dto.BinderLayout, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return h.binderSvc.AddLeaf(ctx, dto.BinderLeafRequest{
		ProfileID:    profileID,
		CollectionID: in.ID,
		Leaf:         in.Body.Leaf,
		Placement:    in.Body.Placement,
	})
}

func (h *BinderHandler) removeLeaf(ctx context.Context, in binderLeafPathInput) (dto.BinderLayout, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderLayout{}, err
	}

	return h.binderSvc.RemoveLeaf(ctx, dto.BinderLeafLookup{ProfileID: profileID, CollectionID: in.ID, Leaf: in.Leaf})
}

func (h *BinderHandler) getPage(ctx context.Context, in binderPageInput) (dto.BinderPage, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderPage{}, err
	}

	return h.binderSvc.GetPage(ctx, dto.BinderPageLookup{ProfileID: profileID, CollectionID: in.ID, Page: in.Page})
}

func (h *BinderHandler) placeCard(ctx context.Context, in binderSlotCardInput) (dto.BinderPage, error) {
	return h.writeSlot(ctx, in.ID, dto.BinderSlotTarget{Page: in.Page, Row: in.Row, Column: in.Column}, dto.BinderSlotCard, in.Body.CardID)
}

func (h *BinderHandler) clearSlot(ctx context.Context, in binderSlotPathInput) (dto.BinderPage, error) {
	return h.writeSlot(ctx, in.ID, dto.BinderSlotTarget{Page: in.Page, Row: in.Row, Column: in.Column}, dto.BinderSlotEmpty, uuid.Nil)
}

func (h *BinderHandler) markUnidentified(ctx context.Context, in binderSlotPathInput) (dto.BinderPage, error) {
	return h.writeSlot(ctx, in.ID, dto.BinderSlotTarget{Page: in.Page, Row: in.Row, Column: in.Column}, dto.BinderSlotUnidentified, uuid.Nil)
}

// writeSlot returns the written Page, so the client does not need a second read.
func (h *BinderHandler) writeSlot(ctx context.Context, collectionID uuid.UUID, target dto.BinderSlotTarget, state string, cardID uuid.UUID) (dto.BinderPage, error) {
	profileID, err := requireProfileID(ctx)
	if err != nil {
		return dto.BinderPage{}, err
	}

	write := dto.BinderSlotWrite{
		ProfileID:    profileID,
		CollectionID: collectionID,
		Target:       target,
		State:        state,
		CardID:       cardID,
	}
	if err := h.binderSvc.WriteSlot(ctx, write); err != nil {
		return dto.BinderPage{}, err
	}

	// Returns the written Page, so the client does not need a second read.
	return h.binderSvc.GetPage(ctx, dto.BinderPageLookup{ProfileID: profileID, CollectionID: collectionID, Page: target.Page})
}

// Routes marks each operation Secured, which only documents the requirement in
// OpenAPI; RegisterRoutes applies the guard that enforces it.
func (h *BinderHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[binderCreateInput, dto.BinderLayout]{
			OperationID: "create-binder",
			Method:      http.MethodPost,
			Path:        "/collections/{id}/binder",
			Summary:     "Turn one of the current user's empty Collections into a binder",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusCreated,
			Secured:     true,
			HandlerFunc: h.create,
		}),
		endpoint.New(endpoint.Endpoint[binderGridInput, dto.BinderLayout]{
			OperationID: "update-binder-grid",
			Method:      http.MethodPut,
			Path:        "/collections/{id}/binder/layout",
			Summary:     "Change the rows and columns of a binder whose Slots are all empty",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.changeGrid,
		}),
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[collectionIDInput]{
			OperationID: "remove-binder",
			Method:      http.MethodDelete,
			Path:        "/collections/{id}/binder",
			Summary:     "Remove a binder's Layout while all its Slots are empty, leaving a plain Collection",
			Tags:        []string{"binder"},
			Secured:     true,
			HandlerFunc: h.remove,
		}),
		endpoint.New(endpoint.Endpoint[binderLeafInput, dto.BinderLayout]{
			OperationID: "add-binder-leaf",
			Method:      http.MethodPost,
			Path:        "/collections/{id}/binder/leaves",
			Summary:     "Add a Leaf before or after another Leaf, renumbering the Pages after it",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusCreated,
			Secured:     true,
			HandlerFunc: h.addLeaf,
		}),
		endpoint.New(endpoint.Endpoint[binderLeafPathInput, dto.BinderLayout]{
			OperationID: "remove-binder-leaf",
			Method:      http.MethodDelete,
			Path:        "/collections/{id}/binder/leaves/{leaf}",
			Summary:     "Remove a Leaf whose Pages hold no card, renumbering the Pages after it",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.removeLeaf,
		}),
		endpoint.New(endpoint.Endpoint[binderPageInput, dto.BinderPage]{
			OperationID: "get-binder-page",
			Method:      http.MethodGet,
			Path:        "/collections/{id}/binder/pages/{page}",
			Summary:     "Read one Page of a binder with its Slots in grid order",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.getPage,
		}),
		endpoint.New(endpoint.Endpoint[binderSlotCardInput, dto.BinderPage]{
			OperationID: "place-binder-card",
			Method:      http.MethodPut,
			Path:        "/collections/{id}/binder/pages/{page}/slots/{row}/{column}",
			Summary:     "Place a Card in a Slot, replacing the Card it held or taking it from Unidentified; returns the Page",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.placeCard,
		}),
		endpoint.New(endpoint.Endpoint[binderSlotPathInput, dto.BinderPage]{
			OperationID: "clear-binder-slot",
			Method:      http.MethodDelete,
			Path:        "/collections/{id}/binder/pages/{page}/slots/{row}/{column}",
			Summary:     "Empty a Slot, removing its Card or Unidentified mark; returns the Page",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.clearSlot,
		}),
		endpoint.New(endpoint.Endpoint[binderSlotPathInput, dto.BinderPage]{
			OperationID: "mark-binder-slot-unidentified",
			Method:      http.MethodPut,
			Path:        "/collections/{id}/binder/pages/{page}/slots/{row}/{column}/unidentified",
			Summary:     "Mark a Slot Unidentified, removing the Card it held; returns the Page",
			Tags:        []string{"binder"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.markUnidentified,
		}),
	}
}
