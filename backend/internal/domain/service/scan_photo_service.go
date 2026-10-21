package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/core/objectstore"
	"github.com/itsLeonB/cardstack/backend/internal/domain/dto"
)

const (
	// scanPhotoUploadTTL is how long a signed upload address stays valid: a
	// browser uploads straight after it asks, so it needs only a few minutes.
	scanPhotoUploadTTL = 5 * time.Minute

	// scanPhotoContentType is the one type a scan photo is stored as; the
	// address signs it, so a PUT of anything else is refused by R2.
	scanPhotoContentType = "image/jpeg"

	scanPhotoNotConfiguredMsg = "scan photo storage is not configured: set SCAN_PHOTO_R2_ACCOUNT_ID, SCAN_PHOTO_R2_ACCESS_KEY_ID, SCAN_PHOTO_R2_SECRET_ACCESS_KEY and SCAN_PHOTO_R2_BUCKET"
)

// ScanPhotoService signs the upload addresses of the private photo bucket that
// keeps the photos of scan corrections (docs/adr/0022).
type ScanPhotoService interface {
	// RequestUploads signs one PUT address per photo, each under a fresh key
	// with the caller's prefix. Without a configured bucket it is refused with
	// apperr.CodePhotoStorageNotConfigured.
	RequestUploads(ctx context.Context, req dto.ScanPhotoUploadRequest) (dto.ScanPhotoUploads, error)
}

type scanPhotoService struct {
	// store is nil when the photo bucket is not configured.
	store objectstore.ObjectStore
}

// NewScanPhotoService builds the service over store, which is nil when the
// photo bucket is not configured.
func NewScanPhotoService(store objectstore.ObjectStore) ScanPhotoService {
	return &scanPhotoService{store: store}
}

func (s *scanPhotoService) RequestUploads(ctx context.Context, req dto.ScanPhotoUploadRequest) (dto.ScanPhotoUploads, error) {
	if s.store == nil {
		return dto.ScanPhotoUploads{}, apperr.WithCode(apperr.ServiceUnavailableError(scanPhotoNotConfiguredMsg), apperr.CodePhotoStorageNotConfigured)
	}

	uploads := make([]dto.ScanPhotoUpload, req.Count)
	for i := range uploads {
		key := req.ProfileID.String() + "/" + uuid.NewString()
		url, expiresAt, err := s.store.PresignPut(ctx, key, scanPhotoContentType, scanPhotoUploadTTL)
		if err != nil {
			return dto.ScanPhotoUploads{}, err
		}
		uploads[i] = dto.ScanPhotoUpload{Key: key, URL: url, ExpiresAt: expiresAt}
	}
	return dto.ScanPhotoUploads{Uploads: uploads}, nil
}
