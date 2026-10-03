package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"github.com/itsLeonB/ungerr"
	"github.com/stretchr/testify/assert"
)

const leakyDSN = "failed to connect: postgres://admin:s3cr3t@dbhost:5432/cards"

type errOutput struct{ Body struct{ OK bool } }

func newErrAPI(t *testing.T, handler func(context.Context, *struct{}) (*errOutput, error)) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t, NewConfig())
	UseRecovery(api)
	huma.Get(api, "/boom", handler)
	return api
}

func captureLogs(t *testing.T) *[]string {
	t.Helper()
	var logged []string
	orig := logRedacted
	logRedacted = func(err error) { logged = append(logged, err.Error()) }
	t.Cleanup(func() { logRedacted = orig })
	return &logged
}

func TestUnknownErrorIsRedactedAndLogged(t *testing.T) {
	logged := captureLogs(t)
	api := newErrAPI(t, func(context.Context, *struct{}) (*errOutput, error) {
		return nil, errors.New(leakyDSN)
	})

	resp := api.Get("/boom")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", resp.Code)
	}
	body := resp.Body.String()
	for _, secret := range []string{"s3cr3t", "dbhost", "postgres://"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaks %q: %s", secret, body)
		}
	}
	if len(*logged) != 1 || !strings.Contains((*logged)[0], "s3cr3t") {
		t.Fatalf("full error not logged: %v", *logged)
	}
}

func TestWrappedUnknownErrorIsRedacted(t *testing.T) {
	captureLogs(t)
	api := newErrAPI(t, func(context.Context, *struct{}) (*errOutput, error) {
		return nil, ungerr.Wrap(errors.New(leakyDSN), "open db")
	})

	resp := api.Get("/boom")
	if resp.Code != http.StatusInternalServerError || strings.Contains(resp.Body.String(), "s3cr3t") {
		t.Fatalf("want redacted 500, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestAppErrorKeepsStatusAndDetails(t *testing.T) {
	logged := captureLogs(t)
	api := newErrAPI(t, func(context.Context, *struct{}) (*errOutput, error) {
		return nil, ungerr.NotFoundError("card not found")
	})

	resp := api.Get("/boom")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "card not found") {
		t.Fatalf("details missing: %s", resp.Body.String())
	}
	if len(*logged) != 0 {
		t.Fatalf("app errors must not be logged as redacted: %v", *logged)
	}
}

func TestWrappedAppErrorIsRecognised(t *testing.T) {
	captureLogs(t)
	api := newErrAPI(t, func(context.Context, *struct{}) (*errOutput, error) {
		return nil, errors.Join(errors.New("ctx"), ungerr.ConflictError("dup"))
	})

	if resp := api.Get("/boom"); resp.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPanicIsRedactedAndLogged(t *testing.T) {
	logged := captureLogs(t)
	api := newErrAPI(t, func(context.Context, *struct{}) (*errOutput, error) {
		panic(leakyDSN)
	})

	resp := api.Get("/boom")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", resp.Code)
	}
	if strings.Contains(resp.Body.String(), "s3cr3t") || strings.Contains(resp.Body.String(), "goroutine") {
		t.Fatalf("panic leaked: %s", resp.Body.String())
	}
	if len(*logged) != 1 || !strings.Contains((*logged)[0], "s3cr3t") {
		t.Fatalf("panic not logged in full: %v", *logged)
	}
}

func TestValidationErrorsUntouched(t *testing.T) {
	captureLogs(t)
	_, api := humatest.New(t, NewConfig())
	huma.Get(api, "/v", func(context.Context, *struct {
		N int `query:"n" minimum:"1"`
	}) (*errOutput, error) {
		return &errOutput{}, nil
	})

	resp := api.Get("/v?n=0")
	if resp.Code != http.StatusUnprocessableEntity || !strings.Contains(resp.Body.String(), "query.n") {
		t.Fatalf("validation response changed: %d %s", resp.Code, resp.Body.String())
	}
}

type echoInput struct {
	ID   uuid.UUID `path:"id"`
	N    int       `query:"n"`
	Body struct {
		Count int `json:"count"`
	}
}

func TestClientErrorsDoNotEchoCallerInput(t *testing.T) {
	const secret = "caller-supplied-secret"
	captureLogs(t)
	_, api := humatest.New(t, NewConfig())
	huma.Post(api, "/e/{id}", func(context.Context, *echoInput) (*errOutput, error) {
		return &errOutput{}, nil
	})

	valid := uuid.NewString()
	cases := map[string]struct {
		path, body string
		status     int
	}{
		"invalid path uuid":   {"/e/" + secret, `{"count":1}`, http.StatusUnprocessableEntity},
		"invalid query int":   {"/e/" + valid + "?n=" + secret, `{"count":1}`, http.StatusUnprocessableEntity},
		"malformed body":      {"/e/" + valid, `{"count": ` + secret, http.StatusBadRequest},
		"wrong body type":     {"/e/" + valid, `{"count":"` + secret + `"}`, http.StatusUnprocessableEntity},
		"unreadable literal":  {"/e/" + valid, secret, http.StatusBadRequest},
		"unexpected body key": {"/e/" + valid, `{"count":1,"` + secret + `":1}`, http.StatusUnprocessableEntity},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resp := api.Post(c.path, "Content-Type: application/json", strings.NewReader(c.body))
			assert.Equal(t, c.status, resp.Code, resp.Body.String())
			assert.NotContains(t, resp.Body.String(), secret)
		})
	}
}

func TestDocsAreServedOutsideProduction(t *testing.T) {
	for _, env := range []string{"debug", "test"} {
		_, api := humatest.New(t, NewServerConfig(env))
		for _, path := range []string{"/docs", "/openapi.json", "/openapi.yaml"} {
			assert.Equal(t, http.StatusOK, api.Get(path).Code, env+" "+path)
		}
	}
}

func TestDocsAreNotServedInProduction(t *testing.T) {
	_, api := humatest.New(t, NewServerConfig("release"))
	for _, path := range []string{"/docs", "/openapi.json", "/openapi.yaml", "/openapi-3.0.json", "/openapi-3.0.yaml"} {
		assert.Equal(t, http.StatusNotFound, api.Get(path).Code, path)
	}
}
