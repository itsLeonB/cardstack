package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
)

// logRedacted records the full, unredacted error whose detail was withheld
// from the client. Swappable so tests can observe it.
var logRedacted = func(err error) {
	logger.Errorf("unclassified error redacted from response: %v", err)
}

// ErrorModel is Huma's error body plus the stable code an AppError may carry
// (apperr). huma.NewError returns it, so Huma documents the code in the
// OpenAPI schema of every error response.
type ErrorModel struct {
	huma.ErrorModel
	Code string `json:"code,omitempty" example:"login_required" doc:"A stable machine-readable code for the failure, set only when a client is meant to branch on it. login_required: this needs a signed-in caller. binder_quantity_locked: a binder's quantities change only by placing or clearing Slots. collection_not_empty: a Layout needs a Collection with no Inventory Entries. binder_layout_locked: rows, columns or the Layout cannot change while a Slot holds a card. binder_leaf_in_use: the Leaf has a card on one of its Pages. binder_max_card_count: a binder's capacity is its Slot count, so max_card_count must stay 0. move_to_binder: a Move's destination is a binder, and a Move chooses no Slot. photo_storage_not_configured: this server has no private photo bucket, so no scan photo upload address can be signed. trade_seat_taken: the Trade's second seat is already claimed. trade_owner_join: the Trade's owner cannot claim its own second seat."`
}

// installErrorClassifier routes Huma's construction hooks through the shared policy: AppErrors
// keep their status and safe detail, other raw errors are logged and dropped
// from the body, and Huma's own request-error details lose the caller input
// they carry (see clientSafeDetail).
func installErrorClassifier() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		return newError(context.Background(), status, msg, errs...)
	}
	huma.NewErrorWithContext = func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
		return newError(ctx.Context(), status, msg, errs...)
	}
}

func newError(ctx context.Context, status int, msg string, errs ...error) huma.StatusError {
	var details []*huma.ErrorDetail
	var code string
	for _, err := range errs {
		if err == nil {
			continue
		}
		if d, ok := err.(huma.ErrorDetailer); ok {
			details = append(details, clientSafeDetail(status, d.ErrorDetail()))
			continue
		}
		var appErr ungerr.AppError
		walkErrors(err, func(cause error) bool {
			var ok bool
			appErr, ok = errors.AsType[ungerr.AppError](cause)
			return ok
		})
		if appErr != nil {
			status, msg, code = appErr.HttpStatus(), appErrorMessage(appErr), apperr.CodeOf(appErr)
			continue
		}
		cause := transportCause(err)
		if cause != nil {
			recordRequestFailure(ctx, cause)
			if errors.Is(cause, context.DeadlineExceeded) {
				unavailable := apperr.ServiceUnavailableError(nil)
				status, msg = unavailable.HttpStatus(), unavailable.Error()
				continue
			}
		} else {
			logRedacted(err)
		}
		if status >= http.StatusInternalServerError {
			status, msg = http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
		}
	}
	return &ErrorModel{
		ErrorModel: huma.ErrorModel{
			Status: status, Title: http.StatusText(status), Detail: msg, Errors: details,
		},
		Code: code,
	}
}

// ungerr v0.5 has no standard Unwrap method, including inside standard or joined errors.
func walkErrors(err error, visit func(error) bool) bool {
	if err == nil {
		return false
	}
	if visit(err) {
		return true
	}
	switch wrapped := err.(type) {
	case *ungerr.UnknownError:
		return walkErrors(ungerr.Unwrap(wrapped), visit)
	case interface{ Unwrap() []error }:
		for _, cause := range wrapped.Unwrap() {
			if walkErrors(cause, visit) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return walkErrors(wrapped.Unwrap(), visit)
	}
	return false
}

// ClassifyError applies the shared policy before Huma serializes a StatusError.
// It returns nil after a response starts so endpoint returns no replacement body.
func ClassifyError(ctx context.Context, err error) error {
	classified := newError(ctx, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err)
	if failure, ok := ctx.Value(requestFailureKey{}).(*requestFailure); ok && failure.responseStarted() {
		return nil
	}
	return classified
}

// echoingMessagePrefixes start the messages Huma builds by appending a
// parser's own error text, which can quote caller input.
var echoingMessagePrefixes = []string{"invalid value: ", "invalid JSON: ", "expected string to be RFC 5322 email: "}

// callerNamedLocations maps the messages Huma attaches to a location named
// by the caller (an unknown body key) to the location that is safe to return.
var callerNamedLocations = map[string]string{
	"unexpected property": "body",
}

// clientSafeDetail strips what Huma copies from the request into a request
// error (ADR-0013): the offending Value, and any message that is parser
// output. Schema-validation messages (422) are fixed templates and stay.
func clientSafeDetail(status int, d *huma.ErrorDetail) *huma.ErrorDetail {
	safe := &huma.ErrorDetail{Location: d.Location, Message: d.Message}
	if status != http.StatusUnprocessableEntity {
		safe.Message = "malformed request"
		return safe
	}
	for _, p := range echoingMessagePrefixes {
		if strings.HasPrefix(d.Message, p) {
			safe.Message = strings.TrimSuffix(p, ": ")
		}
	}
	if loc, ok := callerNamedLocations[d.Message]; ok {
		safe.Location = loc
	}
	return safe
}

func appErrorMessage(e ungerr.AppError) string {
	if d := e.Details(); d != nil {
		return fmt.Sprint(d)
	}
	return e.Error()
}

// UseRecovery converts a handler panic into the same redacted 500 response as
// any other unclassified error, logging the panic value and stack in full. It
// must be installed before operations are registered, since Huma captures
// API-level middleware at Register time.
func UseRecovery(api huma.API) {
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok && transportCause(err) != nil {
					recordRequestFailure(ctx.Context(), transportCause(err))
				} else {
					logRedacted(fmt.Errorf("panic: %v\n%s", rec, debug.Stack()))
				}
				if responseStarted(ctx) {
					return
				}
				if err := huma.WriteErr(api, ctx, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)); err != nil {
					logger.Errorf("writing panic response: %v", err)
				}
			}
		}()
		next(ctx)
	})
}

func responseStarted(ctx huma.Context) bool {
	for {
		if wrapped, ok := ctx.(interface{ Unwrap() huma.Context }); ok {
			ctx = wrapped.Unwrap()
			continue
		}
		if native, ok := ctx.(interface{ Unwrap() *gin.Context }); ok {
			return native.Unwrap().Writer.Written()
		}
		return ctx.Status() != 0
	}
}
