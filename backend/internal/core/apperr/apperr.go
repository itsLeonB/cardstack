// Package apperr gives an ungerr.AppError a stable, machine-readable code
// (ADR-0013). The HTTP error seam copies the code into the response body, so a
// client can branch on it instead of parsing the human-readable message.
package apperr

import (
	"errors"

	"github.com/itsLeonB/ungerr"
)

// CodeLoginRequired answers a Guest request for something only a signed-in
// caller may have: a later page, a rarity, category or tag filter, or facets.
const CodeLoginRequired = "login_required"

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
