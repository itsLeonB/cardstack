package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
)

// logRedacted records the full, unredacted error whose detail was withheld
// from the client. Swappable so tests can observe it.
var logRedacted = func(err error) {
	logger.Errorf("unclassified error redacted from response: %v", err)
}

// installErrorClassifier makes huma.NewError the single seam that decides
// what reaches the client (huma.NewErrorWithContext delegates to it): AppErrors
// keep their status and safe detail, other raw errors are logged and dropped
// from the body, and Huma's own request-error details lose the caller input
// they carry (see clientSafeDetail).
func installErrorClassifier() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		var details []*huma.ErrorDetail
		for _, err := range errs {
			if err == nil {
				continue
			}
			if d, ok := err.(huma.ErrorDetailer); ok {
				details = append(details, clientSafeDetail(status, d.ErrorDetail()))
				continue
			}
			if appErr, ok := errors.AsType[ungerr.AppError](err); ok {
				status = appErr.HttpStatus()
				msg = appErrorMessage(appErr)
				continue
			}
			logRedacted(err)
			if status >= http.StatusInternalServerError {
				status, msg = http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
			}
		}

		return &huma.ErrorModel{
			Status: status,
			Title:  http.StatusText(status),
			Detail: msg,
			Errors: details,
		}
	}
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
				logRedacted(fmt.Errorf("panic: %v\n%s", rec, debug.Stack()))
				if err := huma.WriteErr(api, ctx, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)); err != nil {
					logger.Errorf("writing panic response: %v", err)
				}
			}
		}()
		next(ctx)
	})
}
