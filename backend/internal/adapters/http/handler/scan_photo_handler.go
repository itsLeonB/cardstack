package handler

import (
	"context"
	"net/http"

	authpkg "github.com/itsLeonB/cardstack/backend/internal/adapters/http/auth"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
)

// ScanPhotoHandler serves the scan photo upload addresses (docs/adr/0022).
type ScanPhotoHandler struct {
	photos service.ScanPhotoService
}

func NewScanPhotoHandler(photos service.ScanPhotoService) *ScanPhotoHandler {
	return &ScanPhotoHandler{photos: photos}
}

type requestScanPhotoUploadsInput struct {
	Body struct {
		Count int `json:"count" required:"true" minimum:"1" maximum:"100" doc:"How many photos are to be uploaded. One address comes back per photo."`
	}
}

func (h *ScanPhotoHandler) requestUploads(ctx context.Context, in requestScanPhotoUploadsInput) (dto.ScanPhotoUploads, error) {
	return h.photos.RequestUploads(ctx, dto.ScanPhotoUploadRequest{ProfileID: authpkg.CallerFrom(ctx).ProfileID, Count: in.Body.Count})
}

// Routes returns every route ScanPhotoHandler exposes, for registration via
// endpoint.RegisterAll.
func (h *ScanPhotoHandler) Routes() []endpoint.Registrable {
	return []endpoint.Registrable{
		endpoint.New(endpoint.Endpoint[requestScanPhotoUploadsInput, dto.ScanPhotoUploads]{
			OperationID: "request-scan-photo-uploads",
			Method:      http.MethodPost,
			Path:        "/scan/photo-uploads",
			Summary:     "Sign one PUT address per scan photo, up to 100 in a request, each to a JPEG in the private photo bucket that expires in five minutes. 503 with code photo_storage_not_configured when that bucket is not configured",
			Tags:        []string{"scan"},
			SuccessCode: http.StatusOK,
			Secured:     true,
			HandlerFunc: h.requestUploads,
		}),
	}
}
