package dto

import (
	"time"

	"github.com/google/uuid"
)

// AdminCardUpsert is what an Admin enters for a Card. RarityID is optional:
// without it a new Card gets the promo rarity "Tanpa Tanda" of its Game, and an
// existing Card keeps its rarity. An empty Illustrator likewise keeps the
// existing Card's.
type AdminCardUpsert struct {
	ExpansionSetID uuid.UUID
	LocalID        string
	Name           string
	Category       string
	Tag            string
	RarityID       uuid.UUID
	Illustrator    string
}

// AdminCard is a Card as the admin sees it. ImageURL is empty until an
// uploaded image has been confirmed.
type AdminCard struct {
	ID             uuid.UUID `json:"id"`
	ExpansionSetID uuid.UUID `json:"expansionSetId"`
	LocalID        string    `json:"localId"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	Tag            string    `json:"tag"`
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
	ExpansionSetID uuid.UUID
	LocalID        string
}

// CardImageConfirm names the Card whose uploaded image is to be checked.
type CardImageConfirm struct {
	CardID uuid.UUID
}
