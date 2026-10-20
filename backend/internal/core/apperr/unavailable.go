package apperr

import (
	"fmt"
	"net/http"

	"github.com/itsLeonB/ungerr"
)

// unavailableError is a 503 for a feature the server is not set up to serve
// yet. ungerr has no 503, and a refusal that is not a 500 must say so.
type unavailableError struct {
	details any
}

func (e unavailableError) GrpcStatus() uint32 { return 14 }

func (e unavailableError) HttpStatus() int { return http.StatusServiceUnavailable }

func (e unavailableError) Error() string { return http.StatusText(e.HttpStatus()) }

func (e unavailableError) Details() any { return e.details }

func (e unavailableError) ToLogAttrs() []ungerr.LogAttr {
	return []ungerr.LogAttr{
		{Key: "error.type", Value: "ServiceUnavailableError"},
		{Key: "error.message", Value: fmt.Sprintf("%v", e.details)},
	}
}

// ServiceUnavailableError answers 503 with details as the message.
func ServiceUnavailableError(details any) ungerr.AppError {
	return unavailableError{details}
}
