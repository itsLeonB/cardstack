// Package apperr gives an ungerr.AppError a stable, machine-readable code
// (ADR-0013). The HTTP error seam copies the code into the response body, so a
// client can branch on it instead of parsing the human-readable message.
package apperr

import (
	"errors"
	"net/http"

	"github.com/itsLeonB/ungerr"
)

// CodeLoginRequired answers a Guest request for something only a signed-in
// caller may have: a later page, a rarity, category or tag filter, or facets.
const CodeLoginRequired = "login_required"

// CodeBinderQuantityLocked refuses a quantity write on a binder: its quantities
// change only by placing or clearing Slots (docs/adr/0020).
const CodeBinderQuantityLocked = "binder_quantity_locked"

// CodeCollectionNotEmpty refuses a Layout on a Collection that already holds
// Inventory Entries, which no Slot would explain.
const CodeCollectionNotEmpty = "collection_not_empty"

// CodeBinderLayoutLocked refuses a change to a binder's rows and columns, or
// removing its Layout, while a Slot holds a card.
const CodeBinderLayoutLocked = "binder_layout_locked"

// CodeBinderLeafInUse refuses removing a Leaf with a card on one of its Pages.
const CodeBinderLeafInUse = "binder_leaf_in_use"

// CodeBinderMaxCardCount refuses a Collection's max_card_count on a binder,
// whose capacity is its Slot count.
const CodeBinderMaxCardCount = "binder_max_card_count"

// CodePhotoStorageNotConfigured refuses a scan photo upload address on a server
// whose private photo bucket has no SCAN_PHOTO_R2_* settings (docs/adr/0022).
const CodePhotoStorageNotConfigured = "photo_storage_not_configured"

// coded deliberately embeds the AppError interface rather than the concrete
// ungerr error, so it is not a huma.StatusError: Huma then routes it through
// the classifier in the httpapi package instead of marshalling it directly,
// and the classifier is what puts the code in the body.
type coded struct {
	ungerr.AppError
	code string
}

func (c coded) Code() string { return c.code }

// WithCode returns err carrying code. It is still an ungerr.AppError with the
// same status and message.
func WithCode(err ungerr.AppError, code string) ungerr.AppError {
	return coded{AppError: err, code: code}
}

type codedError interface {
	error
	Code() string
}

// CodeOf returns the code err carries, or "" when it has none.
func CodeOf(err error) string {
	if c, ok := errors.AsType[codedError](err); ok {
		return c.Code()
	}
	return ""
}

// IsNotFound reports whether err is a not-found AppError, the answer a missing
// row or a missing Page gets. It compares the HTTP status the AppError carries.
func IsNotFound(err error) bool {
	appErr, ok := errors.AsType[ungerr.AppError](err)
	return ok && appErr.HttpStatus() == http.StatusNotFound
}
