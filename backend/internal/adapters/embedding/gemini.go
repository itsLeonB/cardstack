// Package embedding holds the production adapters behind the embedding seams in
// core/embedding: the Gemini batch embedder and the hosted-image fetcher.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/ungerr"
	"google.golang.org/genai"
)

// GeminiBatchEmbedder embeds images with Gemini's batch embedding jobs. A job
// reads a JSONL file uploaded with the Files API rather than inline requests,
// because inline requests are capped at 20 MB in total and a few hundred card
// images exceed that. Each line carries the image's key, which the result echoes,
// so vectors are matched by key and never by position.
type GeminiBatchEmbedder struct {
	client *genai.Client
	model  string
}

// NewGeminiBatchEmbedder returns a batch embedder that uses the Gemini Developer
// API with apiKey and embeds with model.
func NewGeminiBatchEmbedder(ctx context.Context, apiKey, model string) (*GeminiBatchEmbedder, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, ungerr.Wrap(err, "creating Gemini client")
	}
	return &GeminiBatchEmbedder{client: client, model: model}, nil
}

// GeminiImageEmbedder embeds one image per call, for a scan that waits on the
// vector. It asks for the same model and dimension as GeminiBatchEmbedder, so
// its vectors are comparable with the stored ones.
type GeminiImageEmbedder struct {
	client *genai.Client
	model  string
}

// NewGeminiImageEmbedder returns an image embedder that uses the Gemini
// Developer API with apiKey and embeds with model.
func NewGeminiImageEmbedder(ctx context.Context, apiKey, model string) (*GeminiImageEmbedder, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, ungerr.Wrap(err, "creating Gemini client")
	}
	return &GeminiImageEmbedder{client: client, model: model}, nil
}

// imageEmbedTimeout bounds one call. It stays under the API's request timeout
// (APP_TIMEOUT, 10s by default) so a slow provider fails here, with our error,
// rather than as a cut-off request. Gemini answered in about 1.5 s at p95.
const imageEmbedTimeout = 8 * time.Second

// Embed returns the normalised vector of one image.
func (g *GeminiImageEmbedder) Embed(ctx context.Context, mimeType string, data []byte) ([]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, imageEmbedTimeout)
	defer cancel()

	dimensions := int32(embedding.Dimensions)
	resp, err := g.client.Models.EmbedContent(ctx, g.model,
		[]*genai.Content{{Parts: []*genai.Part{{InlineData: &genai.Blob{MIMEType: mimeType, Data: data}}}}},
		&genai.EmbedContentConfig{OutputDimensionality: &dimensions})
	if err != nil {
		return nil, ungerr.Wrap(err, "embedding image")
	}
	if len(resp.Embeddings) != 1 || resp.Embeddings[0] == nil {
		return nil, ungerr.Unknownf("embedding image: got %d embeddings, want 1", len(resp.Embeddings))
	}
	values := resp.Embeddings[0].Values
	if len(values) != embedding.Dimensions {
		return nil, ungerr.Unknownf("embedding image: got %d values, want %d", len(values), embedding.Dimensions)
	}
	vector, err := normalise(values)
	if err != nil {
		return nil, ungerr.Wrap(err, "embedding image")
	}
	return vector, nil
}

// batchLine is one request of the input JSONL file.
type batchLine struct {
	Key     string       `json:"key"`
	Request embedRequest `json:"request"`
}

type embedRequest struct {
	Content              *genai.Content `json:"content"`
	OutputDimensionality int            `json:"output_dimensionality"`
}

// resultLine is one line of the output JSONL file: the vector or the error of
// the request with Key.
type resultLine struct {
	Key      string `json:"key"`
	Response *struct {
		Embedding *struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	} `json:"response"`
	Error *genai.JobError `json:"error"`
}

// Submit uploads images as one JSONL file and starts an embedding job over it.
func (g *GeminiBatchEmbedder) Submit(ctx context.Context, images []embedding.Image) (string, error) {
	var input bytes.Buffer
	enc := json.NewEncoder(&input)
	for _, img := range images {
		line := batchLine{
			Key: img.Key,
			Request: embedRequest{
				Content: &genai.Content{Parts: []*genai.Part{
					{InlineData: &genai.Blob{MIMEType: img.MIMEType, Data: img.Data}},
				}},
				OutputDimensionality: embedding.Dimensions,
			},
		}
		if err := enc.Encode(line); err != nil {
			return "", ungerr.Wrapf(err, "encoding image %s", img.Key)
		}
	}

	file, err := g.client.Files.Upload(ctx, &input, &genai.UploadFileConfig{MIMEType: "application/jsonl"})
	if err != nil {
		return "", ungerr.Wrap(err, "uploading embedding batch input")
	}
	job, err := g.client.Batches.CreateEmbeddings(ctx, &g.model, &genai.EmbeddingsBatchJobSource{FileName: file.Name}, nil)
	if err != nil {
		g.deleteInput(ctx, file.Name)
		return "", ungerr.Wrap(err, "creating embedding batch job")
	}
	return job.Name, nil
}

// deleteInput removes an uploaded input that no job will read. It is best
// effort: a failure is logged, so the error that made the input useless is the
// one the caller returns.
func (g *GeminiBatchEmbedder) deleteInput(ctx context.Context, name string) {
	if _, err := g.client.Files.Delete(ctx, name, nil); err != nil {
		logger.Errorf("deleting uploaded embedding input %s: %v", name, err)
	}
}

// Collect reports the status of the job. Once it has succeeded, the result file
// is read and each vector validated and normalised.
func (g *GeminiBatchEmbedder) Collect(ctx context.Context, name string) (embedding.Outcome, error) {
	job, err := g.client.Batches.Get(ctx, name, nil)
	if err != nil {
		return embedding.Outcome{}, ungerr.Wrapf(err, "getting embedding batch job %s", name)
	}

	switch job.State {
	case genai.JobStateSucceeded, genai.JobStatePartiallySucceeded:
		return g.collectResults(ctx, name, job)
	case genai.JobStateFailed, genai.JobStateCancelled, genai.JobStateExpired:
		reason := string(job.State)
		if job.Error != nil && job.Error.Message != "" {
			reason += ": " + job.Error.Message
		}
		return embedding.Outcome{Status: embedding.JobFailed, Reason: reason}, nil
	default:
		return embedding.Outcome{Status: embedding.JobRunning}, nil
	}
}

func (g *GeminiBatchEmbedder) collectResults(ctx context.Context, name string, job *genai.BatchJob) (embedding.Outcome, error) {
	if job.Dest == nil || job.Dest.FileName == "" {
		return embedding.Outcome{}, ungerr.Unknownf("embedding batch job %s finished without a result file", name)
	}
	data, err := g.client.Files.Download(ctx, &genai.File{DownloadURI: job.Dest.FileName}, nil)
	if err != nil {
		return embedding.Outcome{}, ungerr.Wrapf(err, "downloading results of %s", name)
	}

	outcome := embedding.Outcome{
		Status:   embedding.JobSucceeded,
		Vectors:  map[string][]float32{},
		Failures: map[string]string{},
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var line resultLine
		if err := dec.Decode(&line); err != nil {
			return embedding.Outcome{}, ungerr.Wrapf(err, "reading results of %s", name)
		}
		vector, err := resultVector(line)
		if err != nil {
			outcome.Failures[line.Key] = err.Error()
			continue
		}
		outcome.Vectors[line.Key] = vector
	}
	return outcome, nil
}

// resultVector returns the normalised vector of one result line, or the reason
// its image did not embed.
func resultVector(line resultLine) ([]float32, error) {
	if line.Error != nil {
		return nil, errors.New(line.Error.Message)
	}
	if line.Response == nil || line.Response.Embedding == nil {
		return nil, errors.New("result has no embedding")
	}
	values := line.Response.Embedding.Values
	if len(values) != embedding.Dimensions {
		return nil, fmt.Errorf("embedding has %d values, want %d", len(values), embedding.Dimensions)
	}
	return normalise(values)
}

// normalise scales v to unit length, so cosine distance and dot product rank
// the same.
func normalise(v []float32) ([]float32, error) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return nil, errors.New("embedding is a zero vector")
	}

	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out, nil
}
