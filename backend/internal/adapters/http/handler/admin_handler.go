package handler

import (
	"context"
	"net/http"

	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

// OpAdminCheck is the operation id of the Admin probe.
const OpAdminCheck = "admin-check"

// AdminHandler holds the admin routes (docs/adr/0019). RegisterRoutes mounts
// them behind authpkg.AdminOnly, so a handler here never sees a non-Admin.
type AdminHandler struct{}

func NewAdminHandler() *AdminHandler { return &AdminHandler{} }

type AdminCheckInput struct{}

// checkAdmin is empty on purpose: reaching it already proves the caller is an
// Admin, which is all the /admin page asks.
func (h *AdminHandler) checkAdmin(context.Context, AdminCheckInput) error { return nil }

// Routes returns every route AdminHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *AdminHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.NewNoBody(endpoint.NoBodyEndpoint[AdminCheckInput]{
			OperationID: OpAdminCheck,
			Method:      http.MethodGet,
			Path:        "/admin/check",
			Summary:     "Check that the caller is an Admin",
			Tags:        []string{"admin"},
			Secured:     true,
			HandlerFunc: h.checkAdmin,
		}),
	}
}
