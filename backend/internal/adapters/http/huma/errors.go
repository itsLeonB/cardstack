package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/danielgtaylor/huma/v2"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
)

// logRedacted records the full, unredacted error whose detail was withheld
// from the client. Swappable so tests can observe it; nil-safe because
// logger.Global is only initialised in the real server.
var logRedacted = func(err error) {
	logError("unclassified error redacted from response: %v", err)
}

func logError(format string, args ...any) {
	if logger.Global != nil {
		logger.Global.Errorf(format, args...)
	}
}

// installErrorClassifier makes huma.NewError the single seam that decides
// what reaches the client (huma.NewErrorWithContext delegates to it): AppErrors
// keep their status and safe detail, other raw errors at 5xx are logged and
// dropped from the body.
//
// ponytail: raw errors below 500 are Huma-internal request decoding errors
// (out of scope), so they keep Huma's default behaviour.
func installErrorClassifier() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		var details []*huma.ErrorDetail
		for _, err := range errs {
			if err == nil {
				continue
			}
			if d, ok := err.(huma.ErrorDetailer); ok {
				details = append(details, d.ErrorDetail())
				continue
			}
			if appErr, ok := errors.AsType[ungerr.AppError](err); ok {
				status = appErr.HttpStatus()
				msg = appErrorMessage(appErr)
				continue
			}
			if status >= http.StatusInternalServerError {
				logRedacted(err)
				status, msg = http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
				continue
			}
			details = append(details, &huma.ErrorDetail{Message: err.Error()})
		}

		return &huma.ErrorModel{
			Status: status,
			Title:  http.StatusText(status),
			Detail: msg,
			Errors: details,
		}
	}
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
					logError("writing panic response: %v", err)
				}
			}
		}()
		next(ctx)
	})
}
