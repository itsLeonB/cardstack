package dto

import (
	"time"

	"github.com/google/uuid"
)

// ScanPhotoUploadRequest asks for one upload address per photo. ProfileID is
// the caller's, and its prefix is the one every returned key sits under.
type ScanPhotoUploadRequest struct {
	ProfileID uuid.UUID
	Count     int
}

// ScanPhotoUploads holds one signed address per requested photo, in order.
type ScanPhotoUploads struct {
	Uploads []ScanPhotoUpload `json:"uploads"`
}

// ScanPhotoUpload is one photo's address: the browser PUTs a JPEG to URL before
// ExpiresAt, and Key names the object in the feedback batch.
type ScanPhotoUpload struct {
	Key       string    `json:"key"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}
