package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/rs/zerolog"
)

type requestFailureKey struct{}

type requestFailure struct {
	cause           error
	responseStarted func() bool
}

// RequestDeadline asks context-aware work to stop; only the handler writes the response.
func RequestDeadline(timeout time.Duration, log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		failure := &requestFailure{responseStarted: c.Writer.Written}
		ctx = context.WithValue(ctx, requestFailureKey{}, failure)
		c.Request = c.Request.WithContext(ctx)
		defer func() {
			cause := failure.cause
			// Huma can swallow a write error on disconnect. Check before our own cancel.
			if errors.Is(ctx.Err(), context.Canceled) {
				cause = context.Canceled
			}
			if cause != nil {
				log.Warn().Err(cause).Int("status", c.Writer.Status()).Msg(requestFailureMessage(cause))
			}
		}()
		c.Next()
	}
}

func transportCause(err error) error {
	for _, target := range []error{context.Canceled, context.DeadlineExceeded} {
		if walkErrors(err, func(cause error) bool { return errors.Is(cause, target) }) {
			return target
		}
	}
	return nil
}

func requestFailureMessage(cause error) string {
	if errors.Is(cause, context.Canceled) {
		return "request canceled"
	}
	return "request deadline exceeded"
}

func recordRequestFailure(ctx context.Context, cause error) {
	if failure, ok := ctx.Value(requestFailureKey{}).(*requestFailure); ok {
		failure.cause = cause
		return
	}
	logger.Errorf("%s: %v", requestFailureMessage(cause), cause)
}
