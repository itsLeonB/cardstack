// Command tune-scan is an offline report that sets the two provisional scan
// settings from real data: MATCH_EMPTY_THRESHOLD (cells) and MATCH_REPRINT_DELTA
// (reprints). It writes nothing: every database call is a SELECT. It is run by
// hand, never on Railway or in CI.
//
//	make tune-scan ARGS="cells --photos DIR --grid 3x3 [--threshold 0.85] [--cut-dir DIR]"
//	make tune-scan ARGS="reprints [--limit 20] [--min-size 2]"
//
// Both subcommands need the catalog to hold embeddings of the configured
// EMBEDDING_MODEL, and fail before any Gemini call when it does not. cells also
// needs bun on the PATH and the frontend folder beside backend/, because it cuts
// the photos with the app's own cell cut (frontend/tools/cut-pages.ts).
//
// cells cuts each Page photo in DIR into its grid and embeds every cell with
// Gemini. For each cell it prints the top similarity the matcher would report
// for it: 1 minus the cosine distance to the nearest catalog image of the
// configured model, among published Expansion Sets, one row per Card. The line
// also names that top Card and its set code. Compare the top scores of the
// empty cells with those of the filled ones to choose the threshold. Each photo
// then gets a summary: the lowest and
// highest similarity and, with --threshold, how many scored cells are strictly
// below it. Set MATCH_EMPTY_THRESHOLD between the top of the empty cells and the
// bottom of the filled ones. --cut-dir keeps the cut cell JPEGs; otherwise they
// go to a temporary folder that is removed afterwards. Progress goes to the log;
// the report comes at the end on stdout.
//
// reprints groups the Cards by normalized name (the Reprint rule's name test)
// and, for each group of at least --min-size embedded Cards, prints the cosine
// distance of every pair of its Cards. A pair's distance is the nearest pair of
// their embeddings, as ReprintsOf measures it. A pair at or under
// MATCH_REPRINT_DELTA is the same artwork, so that is the Reprint rule's art
// test. Groups print largest first, at most --limit of them, and each Card is
// shown as its set code and local id. A group over 8 Cards gets pairwise
// distances for its first 8 only, and the report says so.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	embeddingadapter "github.com/itsLeonB/cardstack/backend/internal/adapters/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	coreembedding "github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/domain/repository"
	"github.com/itsLeonB/cardstack/backend/internal/provider"
	crud "github.com/itsLeonB/go-crud"
	_ "github.com/joho/godotenv/autoload"
)

const usage = `usage:
  tune-scan cells --photos DIR --grid RxC [--threshold X] [--cut-dir DIR]
  tune-scan reprints [--limit N] [--min-size K]`

// env is what a run needs from outside. main fills it with the real tools,
// database and embedder; tests fill it with fakes.
type env struct {
	stdout, stderr io.Writer
	lookPath       func(string) (string, error)
	// frontendDir is the frontend folder that holds tools/cut-pages.ts.
	frontendDir string
	cut         cutter
	open        func() (deps, func(), error)
	newEmbedder func(context.Context) (coreembedding.ImageEmbedder, error)
}

// deps is the read-only access the reports read through.
type deps struct {
	model        string
	reprintDelta float64
	tuning       repository.TuningRepository
	embeddings   repository.EmbeddingRepository
	cards        repository.MatchRepository
}

func main() {
	logger.Init("TuneScan")

	e := env{
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		lookPath:    exec.LookPath,
		frontendDir: frontendDir(),
		cut:         cutWithBun,
		open:        openDeps,
		newEmbedder: newGeminiEmbedder,
	}
	os.Exit(run(context.Background(), os.Args[1:], e))
}

func run(ctx context.Context, args []string, e env) int {
	if len(args) == 0 {
		printf(e.stderr, "%s\n", usage)
		return 2
	}
	switch args[0] {
	case "cells":
		return runCells(ctx, args[1:], e)
	case "reprints":
		return runReprints(ctx, args[1:], e)
	}
	printf(e.stderr, "tune-scan: unknown command %q\n%s\n", args[0], usage)
	return 2
}

// parseFlags parses a subcommand's flags. ok is false when the run should stop
// with exit, which is 0 for -h and 2 for a bad flag or a stray argument.
func parseFlags(fs *flag.FlagSet, args []string, e env) (exit int, ok bool) {
	err := fs.Parse(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return 0, false
	case err != nil:
		return 2, false
	}
	if fs.NArg() > 0 {
		return usageFail(e, fs, "unexpected argument %q", fs.Arg(0)), false
	}
	return 0, true
}

// usageFail reports a bad flag value with the subcommand's usage, and exits 2.
func usageFail(e env, fs *flag.FlagSet, format string, args ...any) int {
	printf(e.stderr, "tune-scan: "+format+"\n", args...)
	fs.Usage()
	return 2
}

// fail reports a run error in one line and exits 1.
func fail(w io.Writer, err error) int {
	printf(w, "tune-scan: %s\n", err.Error())
	return 1
}

// printf writes a diagnostic line. A failed write to the terminal has nowhere to
// go but the log.
func printf(w io.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		logger.Errorf("writing output: %v", err)
	}
}

// report is the stdout report. It keeps the first write error, so the lines read
// straight through and the run checks once, at the end.
type report struct {
	w   io.Writer
	err error
}

func (r *report) printf(format string, args ...any) {
	if r.err == nil {
		_, r.err = fmt.Fprintf(r.w, format, args...)
	}
}

// requireEmbeddings stops a report before anything is embedded when the catalog
// has no vectors of the configured model to compare with.
func requireEmbeddings(ctx context.Context, d deps) error {
	has, err := d.tuning.HasEmbeddings(ctx, d.model)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("the catalog has no embeddings for model %q: run make embed-catalog and make collect-embeddings first, or check EMBEDDING_MODEL", d.model)
	}
	return nil
}

// openDeps loads the configuration and opens the database. The returned cleanup
// closes it. It builds no embedder, so reprints never needs a Gemini key.
func openDeps() (deps, func(), error) {
	if err := config.Load(); err != nil {
		return deps{}, nil, err
	}
	ds, cleanup, err := provider.ProvideDataSource()
	if err != nil {
		return deps{}, nil, err
	}
	return deps{
		model:        config.Global.Model,
		reprintDelta: config.Global.ReprintDelta,
		tuning:       repository.NewTuningRepository(crud.NewRepository[entity.Card](ds.Gorm)),
		embeddings:   repository.NewEmbeddingRepository(crud.NewRepository[entity.CardEmbedding](ds.Gorm)),
		cards:        repository.NewMatchRepository(crud.NewRepository[entity.Card](ds.Gorm)),
	}, cleanup, nil
}

// newGeminiEmbedder builds the same Gemini embedder the scan match uses, from
// the same settings, so the cells are embedded the way the API embeds a photo.
func newGeminiEmbedder(ctx context.Context) (coreembedding.ImageEmbedder, error) {
	if config.Global.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set: cells embeds each cell with Gemini")
	}
	return embeddingadapter.NewGeminiImageEmbedder(ctx, config.Global.APIKey, config.Global.Model)
}

// frontendDir is the frontend folder beside the backend module: the nearest
// folder above the working directory that holds go.mod, plus ../frontend. It is
// empty when there is no go.mod above, and cells then reports the cutter missing.
func frontendDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "..", "frontend")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
