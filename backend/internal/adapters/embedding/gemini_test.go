package embedding

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

// fakeGemini answers the Gemini Developer API calls the batch embedder makes:
// the resumable file upload, batch create and get, and the result download.
type fakeGemini struct {
	srv       *httptest.Server
	uploaded  []byte
	createdAt string
	jobState  string
	// noOutput makes a finished job report no result file.
	noOutput bool
	results  string
	// createStatus, when set, makes the batch create call fail with that status.
	createStatus int
	// deleteStatus, when set, makes the file delete call fail with that status.
	deleteStatus int
	// deleted records that the uploaded input file was deleted.
	deleted bool
}

func newFakeGemini(t *testing.T) *fakeGemini {
	t.Helper()
	f := &fakeGemini{}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeGemini) serve(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch {
	case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/files/input-1"):
		f.deleted = true
		if f.deleteStatus != 0 {
			http.Error(w, "delete refused", f.deleteStatus)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	case r.Header.Get("X-Goog-Upload-Command") == "start":
		w.Header().Set("X-Goog-Upload-URL", f.srv.URL+"/upload-target")
	case r.URL.Path == "/upload-target":
		f.uploaded = body
		w.Header().Set("X-Goog-Upload-Status", "final")
		_, _ = w.Write([]byte(`{"file":{"name":"files/input-1","uri":"` + f.srv.URL + `/files/input-1"}}`))
	case strings.HasSuffix(r.URL.Path, ":asyncBatchEmbedContent"):
		if f.createStatus != 0 {
			http.Error(w, "create refused", f.createStatus)
			return
		}
		f.createdAt = string(body)
		_, _ = w.Write([]byte(`{"name":"batches/job-1","metadata":{"state":"JOB_STATE_PENDING"}}`))
	case strings.HasSuffix(r.URL.Path, "/batches/job-1"):
		output := `,"output":{"responsesFile":"files/output-1"}`
		if f.noOutput {
			output = ""
		}
		_, _ = w.Write([]byte(`{"name":"batches/job-1","metadata":{"state":"` + f.jobState + `"` + output + `}}`))
	case strings.HasSuffix(r.URL.Path, "/files/output-1:download"):
		_, _ = w.Write([]byte(f.results))
	default:
		http.NotFound(w, r)
	}
}

func newTestEmbedder(t *testing.T, f *fakeGemini) *GeminiBatchEmbedder {
	t.Helper()
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:      "test-key",
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{BaseURL: f.srv.URL},
	})
	require.NoError(t, err)
	return &GeminiBatchEmbedder{client: client, model: "gemini-embedding-2"}
}

// axisVector is a vector of the configured dimension with value on one axis.
func axisVector(axis int, value float32) []float32 {
	v := make([]float32, embedding.Dimensions)
	v[axis] = value
	return v
}

// vectorResult is one output line whose embedding is values.
func vectorResult(t *testing.T, key string, values []float32) string {
	t.Helper()
	line, err := json.Marshal(map[string]any{
		"key":      key,
		"response": map[string]any{"embedding": map[string]any{"values": values}},
	})
	require.NoError(t, err)
	return string(line)
}

// errorResult is one output line whose request failed with message.
func errorResult(t *testing.T, key, message string) string {
	t.Helper()
	line, err := json.Marshal(map[string]any{"key": key, "error": map[string]any{"message": message}})
	require.NoError(t, err)
	return string(line)
}

func TestGeminiBatchEmbedder_SubmitUploadsOneLinePerImageAndCreatesTheJob(t *testing.T) {
	f := newFakeGemini(t)
	e := newTestEmbedder(t, f)

	job, err := e.Submit(context.Background(), []embedding.Image{
		{Key: "card-a", MIMEType: "image/png", Data: []byte("png-a")},
		{Key: "card-b", MIMEType: "image/jpeg", Data: []byte("jpeg-b")},
	})
	require.NoError(t, err)
	assert.Equal(t, "batches/job-1", job)

	lines := strings.Split(strings.TrimSpace(string(f.uploaded)), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"output_dimensionality":1536`)
	assert.Contains(t, lines[0], `"mimeType":"image/png"`)

	var first batchLine
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
	assert.Equal(t, "card-a", first.Key)
	assert.Equal(t, []byte("png-a"), first.Request.Content.Parts[0].InlineData.Data)

	assert.Contains(t, f.createdAt, "files/input-1", "the job reads the uploaded file")
}

func TestGeminiBatchEmbedder_CollectReportsAnUnfinishedJobAsRunning(t *testing.T) {
	for _, state := range []string{"JOB_STATE_PENDING", "JOB_STATE_RUNNING"} {
		t.Run(state, func(t *testing.T) {
			f := newFakeGemini(t)
			f.jobState = state
			e := newTestEmbedder(t, f)

			outcome, err := e.Collect(context.Background(), "batches/job-1")
			require.NoError(t, err)
			assert.Equal(t, embedding.JobRunning, outcome.Status)
			assert.Empty(t, outcome.Vectors)
		})
	}
}

func TestGeminiBatchEmbedder_CollectReportsFailedJobsAsFailed(t *testing.T) {
	for _, state := range []string{"JOB_STATE_FAILED", "JOB_STATE_CANCELLED", "JOB_STATE_EXPIRED"} {
		t.Run(state, func(t *testing.T) {
			f := newFakeGemini(t)
			f.jobState = state
			e := newTestEmbedder(t, f)

			outcome, err := e.Collect(context.Background(), "batches/job-1")
			require.NoError(t, err)
			assert.Equal(t, embedding.JobFailed, outcome.Status)
			assert.Contains(t, outcome.Reason, state)
		})
	}
}

func TestGeminiBatchEmbedder_CollectReturnsNormalisedVectorsAndPerItemFailures(t *testing.T) {
	for _, state := range []string{"JOB_STATE_SUCCEEDED", "JOB_STATE_PARTIALLY_SUCCEEDED"} {
		t.Run(state, func(t *testing.T) {
			f := newFakeGemini(t)
			f.jobState = state
			f.results = strings.Join([]string{
				vectorResult(t, "card-a", axisVector(0, 2)),
				errorResult(t, "card-b", "image unreadable"),
				vectorResult(t, "card-c", make([]float32, 3)),
				vectorResult(t, "card-d", make([]float32, embedding.Dimensions)),
			}, "\n")
			e := newTestEmbedder(t, f)

			outcome, err := e.Collect(context.Background(), "batches/job-1")
			require.NoError(t, err)
			assert.Equal(t, embedding.JobSucceeded, outcome.Status)
			require.Len(t, outcome.Vectors, 1)
			assert.InDelta(t, 1, outcome.Vectors["card-a"][0], 1e-6, "the vector comes back unit length")
			assert.Equal(t, map[string]string{
				"card-b": "image unreadable",
				"card-c": "embedding has 3 values, want 1536",
				"card-d": "embedding is a zero vector",
			}, outcome.Failures)
		})
	}
}

func TestGeminiBatchEmbedder_CollectFailsAJobWithNoResultFile(t *testing.T) {
	f := newFakeGemini(t)
	f.jobState = "JOB_STATE_SUCCEEDED"
	f.noOutput = true
	e := newTestEmbedder(t, f)

	_, err := e.Collect(context.Background(), "batches/job-1")
	require.Error(t, err)
}

func TestGeminiBatchEmbedder_SubmitDeletesTheUploadWhenTheJobIsRefused(t *testing.T) {
	f := newFakeGemini(t)
	f.createStatus = http.StatusBadRequest
	e := newTestEmbedder(t, f)

	_, err := e.Submit(context.Background(), []embedding.Image{{Key: "card-a", MIMEType: "image/png", Data: []byte("png-a")}})
	require.ErrorContains(t, err, "creating embedding batch job")
	assert.True(t, f.deleted, "the input nobody will read is removed")
}

func TestGeminiBatchEmbedder_SubmitKeepsTheUploadOnceTheJobExists(t *testing.T) {
	f := newFakeGemini(t)
	e := newTestEmbedder(t, f)

	_, err := e.Submit(context.Background(), []embedding.Image{{Key: "card-a", MIMEType: "image/png", Data: []byte("png-a")}})
	require.NoError(t, err)
	assert.False(t, f.deleted, "the job still reads its input, so it must stay")
}

func TestGeminiBatchEmbedder_SubmitReportsTheRefusalWhenTheCleanupFails(t *testing.T) {
	f := newFakeGemini(t)
	f.createStatus = http.StatusBadRequest
	f.deleteStatus = http.StatusNotFound
	e := newTestEmbedder(t, f)

	_, err := e.Submit(context.Background(), []embedding.Image{{Key: "card-a", MIMEType: "image/png", Data: []byte("png-a")}})
	require.ErrorContains(t, err, "creating embedding batch job", "a failed cleanup does not mask the refusal")
	assert.True(t, f.deleted)
}

func newTestImageEmbedder(t *testing.T, srv *httptest.Server) *GeminiImageEmbedder {
	t.Helper()
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:      "test-key",
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{BaseURL: srv.URL},
	})
	require.NoError(t, err)
	return &GeminiImageEmbedder{client: client, model: "gemini-embedding-2"}
}

func TestGeminiImageEmbedder_EmbedSendsOneImageAndNormalisesTheVector(t *testing.T) {
	var sent []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent, _ = io.ReadAll(r.Body)
		values, err := json.Marshal(axisVector(3, 4))
		require.NoError(t, err)
		_, _ = w.Write([]byte(`{"embeddings":[{"values":` + string(values) + `}]}`))
	}))
	t.Cleanup(srv.Close)

	vector, err := newTestImageEmbedder(t, srv).Embed(context.Background(), "image/jpeg", []byte("jpeg-bytes"))

	require.NoError(t, err)
	require.Len(t, vector, embedding.Dimensions)
	assert.InDelta(t, 1, vector[3], 1e-6)
	assert.Contains(t, string(sent), `"outputDimensionality":1536`)
	assert.Contains(t, string(sent), `"mimeType":"image/jpeg"`)
}

func TestGeminiImageEmbedder_EmbedFailures(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"provider error": func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "quota", http.StatusTooManyRequests) },
		"no embedding":   func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"embeddings":[]}`)) },
		"short vector": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"embeddings":[{"values":[1,2,3]}]}`))
		},
	}
	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			t.Cleanup(srv.Close)

			_, err := newTestImageEmbedder(t, srv).Embed(context.Background(), "image/jpeg", []byte("x"))

			assert.Error(t, err)
		})
	}
}

func TestGeminiImageEmbedder_EmbedStopsWhenTheCallerGivesUp(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { <-release }))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // runs before srv.Close, which waits for the handler
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	t.Cleanup(cancel)

	start := time.Now()
	_, err := newTestImageEmbedder(t, srv).Embed(ctx, "image/jpeg", []byte("x"))

	assert.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second)
}
