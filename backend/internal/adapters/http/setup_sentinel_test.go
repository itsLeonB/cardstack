package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	httpapi "github.com/itsLeonB/cardstack/backend/internal/adapters/http/huma"
	"github.com/itsLeonB/cardstack/backend/internal/core/apperr"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/endpoint"
	ezzerolog "github.com/itsLeonB/ezutil/v2/zerolog"
	"github.com/itsLeonB/ungerr"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersAreOnEveryResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(securityHeaders())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	want := map[string]string{
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
		"Referrer-Policy":           "no-referrer",
		"X-Frame-Options":           "DENY",
		"Content-Security-Policy":   "frame-ancestors 'none'",
		"X-Content-Type-Options":    "nosniff",
	}

	for path, status := range map[string]int{"/ok": http.StatusOK, "/missing": http.StatusNotFound} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

		require.Equal(t, status, w.Code, path)
		for header, value := range want {
			assert.Equal(t, value, w.Header().Get(header), path+" "+header)
		}
	}
}

func TestSetupSentinel_AllowsTheOriginsOfBothLists(t *testing.T) {
	previous := config.Global
	t.Cleanup(func() { config.Global = previous })
	config.Global = &config.Config{
		App:   config.App{ClientUrls: []string{"https://www.example.com"}, Timeout: time.Second},
		Admin: config.Admin{AdminClientUrls: []string{"https://admin.example.com"}},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	require.NoError(t, setupSentinel(r, nil, zerolog.Nop()))
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, origin := range []string{"https://www.example.com", "https://admin.example.com"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/ok", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		r.ServeHTTP(w, req)

		assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
}

func deadlineAPI(t *testing.T, timeout time.Duration) (*gin.Engine, huma.API, *bytes.Buffer) {
	t.Helper()
	previousConfig, previousLogger := config.Global, logger.Global
	t.Cleanup(func() {
		config.Global, logger.Global = previousConfig, previousLogger
	})
	config.Global = &config.Config{App: config.App{Timeout: timeout}}
	logs := new(bytes.Buffer)
	logger.Global = ezzerolog.NewZerologAdapter(logs)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	require.NoError(t, setupSentinel(router, nil, zerolog.New(logs)))
	api := humagin.New(router, httpapi.NewConfig())
	httpapi.UseRecovery(api)
	return router, api, logs
}

func TestSetupSentinel_DeadlineErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		detail string
		code   string
	}{
		{"raw", context.DeadlineExceeded, 503, "Service Unavailable", ""},
		{"standard", fmt.Errorf("private: %w", context.DeadlineExceeded), 503, "Service Unavailable", ""},
		{"ungerr", ungerr.Wrap(context.DeadlineExceeded, "private"), 503, "Service Unavailable", ""},
		{"mixed", fmt.Errorf("private: %w", ungerr.Wrap(fmt.Errorf("inner: %w", context.DeadlineExceeded), "private")), 503, "Service Unavailable", ""},
		{"joined", errors.Join(errors.New("private"), ungerr.Wrap(context.DeadlineExceeded, "private")), 503, "Service Unavailable", ""},
		{"explicit", ungerr.Wrap(apperr.WithCode(ungerr.ConflictError("safe"), "existing_code"), "private"), 409, "safe", "existing_code"},
		{"explicit before deadline", errors.Join(context.DeadlineExceeded, ungerr.ConflictError("safe")), 409, "safe", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, api, logs := deadlineAPI(t, time.Second)
			registerDeadlineEndpoints(api, func(context.Context) (string, error) { return "", tc.err })
			for _, path := range []string{"/body", "/meta", "/list", "/empty", "/redirect"} {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				require.Equal(t, tc.status, response.Code, response.Body.String())
				var body map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body), "one JSON response")
				assert.Equal(t, tc.detail, body["detail"])
				if tc.code == "" {
					assert.NotContains(t, body, "code")
				} else {
					assert.Equal(t, tc.code, body["code"])
				}
				assert.NotContains(t, response.Body.String(), "private")
				assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
				if tc.status == 503 {
					assert.Contains(t, logs.String(), "request deadline exceeded")
					assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")))
				} else {
					assert.Empty(t, logs.String())
				}
				logs.Reset()
			}
		})
	}
}

func registerDeadlineEndpoints(api huma.API, handler func(context.Context) (string, error)) {
	endpoint.Register(api, endpoint.Endpoint[struct{}, string]{
		Method: http.MethodGet, Path: "/body", OperationID: "body",
		HandlerFunc: func(ctx context.Context, _ struct{}) (string, error) { return handler(ctx) },
	})
	endpoint.RegisterWithMeta(api, endpoint.EndpointWithMeta[struct{}, string, struct{}]{
		Method: http.MethodGet, Path: "/meta", OperationID: "meta",
		HandlerFunc: func(ctx context.Context, _ struct{}) (string, struct{}, error) {
			result, err := handler(ctx)
			return result, struct{}{}, err
		},
	})
	endpoint.RegisterList(api, endpoint.ListEndpoint[struct{}, string]{
		Method: http.MethodGet, Path: "/list", OperationID: "list",
		HandlerFunc: func(ctx context.Context, _ struct{}) ([]string, error) {
			result, err := handler(ctx)
			return []string{result}, err
		},
	})
	endpoint.RegisterNoBody(api, endpoint.NoBodyEndpoint[struct{}]{
		Method: http.MethodGet, Path: "/empty", OperationID: "empty",
		HandlerFunc: func(ctx context.Context, _ struct{}) error { _, err := handler(ctx); return err },
	})
	endpoint.RegisterRedirect(api, endpoint.RedirectEndpoint[struct{}]{
		Method: http.MethodGet, Path: "/redirect", OperationID: "redirect",
		HandlerFunc: func(ctx context.Context, _ struct{}) (string, error) { return handler(ctx) },
	})
}

func TestSetupSentinel_RequestLifetime(t *testing.T) {
	for _, tc := range []struct {
		name   string
		wait   bool
		err    error
		status int
		log    string
	}{
		{"normal", false, nil, 200, ""},
		{"cooperative deadline", true, context.DeadlineExceeded, 503, "request deadline exceeded"},
		{"late success", true, nil, 200, ""},
		{"unrelated error after deadline", true, errors.New("private database failure"), 500, "unclassified error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				router, api, logs := deadlineAPI(t, time.Second)
				var requestContext context.Context
				finished := false
				registerDeadlineEndpoints(api, func(ctx context.Context) (string, error) {
					requestContext = ctx
					if tc.wait {
						<-ctx.Done()
					}
					finished = true
					return "success", tc.err
				})
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/body", nil))

				assert.True(t, finished, "ServeHTTP must not return while the handler is running")
				assert.Error(t, requestContext.Err(), "middleware must release the deadline timer")
				assert.Equal(t, tc.status, response.Code, response.Body.String())
				if tc.status == 200 {
					assert.JSONEq(t, `{"data":"success"}`, response.Body.String())
				}
				if tc.log == "" {
					assert.Empty(t, logs.String())
				} else {
					assert.Contains(t, logs.String(), tc.log)
					assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")))
				}
			})
		})
	}
}

type interruptedResponse struct{}

func (interruptedResponse) MarshalJSON() ([]byte, error) {
	return nil, context.Canceled
}

func TestSetupSentinel_StartedResponse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		panic bool
	}{
		{"cancellation while serializing", false},
		{"panic after native Gin write", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, api, logs := deadlineAPI(t, time.Second)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
				_, err := humagin.Unwrap(ctx).Writer.Write([]byte("partial"))
				require.NoError(t, err)
				if !tc.panic {
					cancel()
				}
				next(ctx)
			})
			endpoint.Register(api, endpoint.Endpoint[struct{}, interruptedResponse]{
				Method: http.MethodGet, Path: "/started", OperationID: "started",
				HandlerFunc: func(context.Context, struct{}) (interruptedResponse, error) {
					if tc.panic {
						panic("private panic")
					}
					return interruptedResponse{}, nil
				},
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequestWithContext(ctx, http.MethodGet, "/started", nil))

			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "partial", response.Body.String(), "no replacement error JSON")
			assert.Equal(t, "nosniff", response.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")), logs.String())
			if tc.panic {
				assert.Contains(t, logs.String(), "private panic")
			} else {
				assert.Contains(t, logs.String(), "request canceled")
				assert.NotContains(t, logs.String(), "request deadline exceeded")
			}
		})
	}
}

func TestSetupSentinel_DeadlineAfterStartedResponse(t *testing.T) {
	router, api, logs := deadlineAPI(t, time.Second)
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		_, err := humagin.Unwrap(ctx).Writer.Write([]byte("partial"))
		require.NoError(t, err)
		next(ctx)
	})
	registerDeadlineEndpoints(api, func(context.Context) (string, error) {
		return "", context.DeadlineExceeded
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/body", nil))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "partial", response.Body.String())
	assert.Contains(t, logs.String(), "request deadline exceeded")
	assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")))
}

func TestSetupSentinel_ReturnedCancellation(t *testing.T) {
	router, api, logs := deadlineAPI(t, time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	registerDeadlineEndpoints(api, func(ctx context.Context) (string, error) {
		cancel()
		return "", ungerr.Wrap(ctx.Err(), "private")
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(ctx, http.MethodGet, "/body", nil))

	assert.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")), logs.String())
	assert.Contains(t, logs.String(), "request canceled")
	assert.NotContains(t, logs.String(), "request deadline exceeded")
	assert.NotContains(t, response.Body.String(), "private")
}

func TestSetupSentinel_DeadlineDocumentation(t *testing.T) {
	_, api, _ := deadlineAPI(t, time.Second)
	registerDeadlineEndpoints(api, func(context.Context) (string, error) { return "", nil })
	for _, path := range []string{"/body", "/meta", "/list", "/empty", "/redirect"} {
		responses := api.OpenAPI().Paths[path].Get.Responses
		require.Contains(t, responses, "503", path)
		assert.Contains(t, responses, "default", "keep documentation for other errors")
		assert.Contains(t, responses["503"].Description, "deadline")
		assert.Equal(t, "#/components/schemas/ErrorModel",
			responses["503"].Content["application/problem+json"].Schema.Ref)
	}
}
