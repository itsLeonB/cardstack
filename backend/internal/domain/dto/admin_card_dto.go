package dto

import (
	"time"

	"github.com/google/uuid"
)

// AdminCardCreate is what an Admin enters for a new Card. RarityID is
// optional: without it the Card gets the promo rarity "Tanpa Tanda" of its
// Game.
type AdminCardCreate struct {
	ExpansionSetID uuid.UUID
	LocalID        string
	Name           string
	Category       string
	Tags           []string
	RarityID       uuid.UUID
	Illustrator    string
}

// AdminCardUpdate replaces every editable field of the Card with ID. The set
// and collector number are the Card's identity and stay as they are. RarityID
// is required and an empty Illustrator clears it: nothing is kept from the
// stored Card.
type AdminCardUpdate struct {
	ID          uuid.UUID
	Name        string
	Category    string
	Tags        []string
	RarityID    uuid.UUID
	Illustrator string
}

// AdminCard is a Card as the admin sees it. ImageURL is empty until an
// uploaded image has been confirmed.
type AdminCard struct {
	ID             uuid.UUID `json:"id"`
	ExpansionSetID uuid.UUID `json:"expansionSetId"`
	LocalID        string    `json:"localId"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	Tags           []string  `json:"tags"`
	RarityID       uuid.UUID `json:"rarityId"`
	Illustrator    string    `json:"illustrator"`
	ImageURL       string    `json:"imageUrl"`
}

// CardImageUploadRequest asks for a signed address to PUT a Card's image to.
type CardImageUploadRequest struct {
	CardID      uuid.UUID
	ContentType string
}

// CardImageUpload is the signed address: the browser must PUT the file to URL
// with a Content-Type header equal to ContentType before ExpiresAt.
type CardImageUpload struct {
	URL         string    `json:"url"`
	ContentType string    `json:"contentType"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// CardLookup names the Card an Admin looks up by its set and collector number.
type CardLookup struct {
	ExpansionSetID uuid.UUID
	LocalID        string
}

// CardDelete names the Card an Admin deletes.
type CardDelete struct {
	ID uuid.UUID
}

// CardImageConfirm names the Card whose uploaded image is to be checked.
type CardImageConfirm struct {
	CardID uuid.UUID
}
