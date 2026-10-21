package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	crud "github.com/itsLeonB/go-crud"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeEmbedder answers each cell, by its bytes, with the vector the test gave
// it, and counts its calls. The real Gemini embedder is never reached.
type fakeEmbedder struct {
	vectors map[string][]float32
	calls   int
}

func (f *fakeEmbedder) Embed(_ context.Context, mimeType string, data []byte) ([]float32, error) {
	f.calls++
	if mimeType != jpegMIME {
		return nil, fmt.Errorf("fake embedder: got %s, want %s", mimeType, jpegMIME)
	}
	v, ok := f.vectors[string(data)]
	if !ok {
		return nil, errors.New("fake embedder: no vector for cell " + string(data))
	}
	return v, nil
}

// harness runs the command in-process over the test database, with the cutter,
// the embedder and the tools faked. Every test starts from newHarness.
type harness struct {
	env      env
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
	embedder *fakeEmbedder
	opens    int
	cuts     int
	cutDirs  []string
}

// newHarness runs the command against db with model as the configured
// embedding model. vectors is keyed by cell, as "<photo>/r<RR>c<CC>".
func newHarness(t *testing.T, db *gorm.DB, model string, vectors map[string][]float32) *harness {
	t.Helper()
	h := &harness{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}, embedder: &fakeEmbedder{vectors: vectors}}

	frontend := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(frontend, "tools"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(frontend, "tools", "cut-pages.ts"), nil, 0o644))

	h.env = env{
		stdout:      h.stdout,
		stderr:      h.stderr,
		lookPath:    func(string) (string, error) { return "/usr/bin/bun", nil },
		frontendDir: frontend,
		cut: func(_ context.Context, _, photosDir, outDir string, g grid) error {
			h.cuts++
			h.cutDirs = append(h.cutDirs, outDir)
			return fakeCut(photosDir, outDir, g)
		},
		open: func() (deps, func(), error) {
			h.opens++
			return deps{
				model:        model,
				reprintDelta: 0.05,
				tuning:       repository.NewTuningRepository(crud.NewRepository[entity.Card](db)),
				embeddings:   repository.NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](db)),
				cards:        repository.NewMatchRepository(crud.NewRepository[entity.Card](db)),
			}, func() {}, nil
		},
		newEmbedder: func(context.Context) (embedding.ImageEmbedder, error) { return h.embedder, nil },
	}
	return h
}

func (h *harness) run(args ...string) int {
	return run(context.Background(), args, h.env)
}

// fakeCut is the cutter the tests use. It cuts every .jpg in photosDir into the
// grid, and the bytes of each cell name it, "<photo>/r<RR>c<CC>", so the fake
// embedder can answer it.
func fakeCut(photosDir, outDir string, g grid) error {
	photos, err := os.ReadDir(photosDir)
	if err != nil {
		return err
	}
	for _, p := range photos {
		photo := strings.TrimSuffix(p.Name(), ".jpg")
		dir := filepath.Join(outDir, photo)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for r := 1; r <= g.rows; r++ {
			for c := 1; c <= g.columns; c++ {
				cell := fmt.Sprintf("r%02dc%02d", r, c)
				if err := os.WriteFile(filepath.Join(dir, cell+".jpg"), []byte(photo+"/"+cell), 0o644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// dirWithPhotos makes a folder holding one .jpg per name. fakeCut reads only the
// names, so the files need no picture in them.
func dirWithPhotos(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("photo"), 0o644))
	}
	return dir
}

// lines is the output as one string per line, with each line's runs of spaces
// collapsed, so a test reads the report without tabwriter's padding. Blank lines
// are dropped.
func lines(out string) []string {
	var got []string
	for _, l := range strings.Split(out, "\n") {
		if fields := strings.Fields(l); len(fields) > 0 {
			got = append(got, strings.Join(fields, " "))
		}
	}
	return got
}

func TestRun_NoOrUnknownCommandIsAUsageError(t *testing.T) {
	db := testDB(t)
	h := newHarness(t, db, "unused", nil)

	require.Equal(t, 2, h.run())
	require.Contains(t, h.stderr.String(), "usage:")

	h.stderr.Reset()
	require.Equal(t, 2, h.run("bogus"))
	require.Contains(t, h.stderr.String(), `unknown command "bogus"`)
	require.Equal(t, 0, h.opens, "no database is opened for a usage error")
}
