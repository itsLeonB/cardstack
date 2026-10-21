package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/google/uuid"
	coreembedding "github.com/itsLeonB/cardstack/backend/internal/core/embedding"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"github.com/itsLeonB/cardstack/backend/internal/domain/service"
)

// jpegMIME is the type of a cut cell, the type the matcher embeds.
const jpegMIME = "image/jpeg"

var cellFileName = regexp.MustCompile(`^r(\d+)c(\d+)\.jpg$`)

// cutCell is one cut cell file. Row and column are 1-based, as its name.
type cutCell struct {
	photo  string
	row    int
	column int
	path   string
}

// cellScore is one cell's report: its top similarity and the Card it is for.
// scored is false when no published Card is embedded to compare with.
type cellScore struct {
	cutCell
	scored     bool
	similarity float64
	cardName   string
	setCode    string
}

func runCells(ctx context.Context, args []string, e env) int {
	fs := flag.NewFlagSet("cells", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	fs.Usage = func() {
		printf(e.stderr, "usage: tune-scan cells --photos DIR --grid RxC [--threshold X] [--cut-dir DIR]\n")
		fs.PrintDefaults()
	}
	photos := fs.String("photos", "", "folder of Page photos (.jpg or .jpeg)")
	gridFlag := fs.String("grid", "", "rows x columns of each Page, like 3x3")
	threshold := fs.Float64("threshold", 0, "count the cells whose top similarity is below this (0 to 1); omit to print only the lowest and highest")
	cutDir := fs.String("cut-dir", "", "keep the cut cell photos in this folder (default: a temporary folder, removed afterwards)")
	if exit, ok := parseFlags(fs, args, e); !ok {
		return exit
	}

	if *photos == "" {
		return usageFail(e, fs, "--photos is required")
	}
	g, err := parseGrid(*gridFlag)
	if err != nil {
		return usageFail(e, fs, "%v", err)
	}
	counting := false
	fs.Visit(func(f *flag.Flag) { counting = counting || f.Name == "threshold" })
	if counting && (*threshold < 0 || *threshold > 1) {
		return usageFail(e, fs, "--threshold must be between 0 and 1")
	}
	photosDir, err := filepath.Abs(*photos)
	if err != nil {
		return fail(e.stderr, fmt.Errorf("resolving --photos: %w", err))
	}
	if info, err := os.Stat(photosDir); err != nil || !info.IsDir() {
		return fail(e.stderr, fmt.Errorf("photos folder not found: %s", photosDir))
	}

	// Tools and inputs are checked before the database is opened or any cell is cut.
	if _, err := e.lookPath("bun"); err != nil {
		return fail(e.stderr, fmt.Errorf("bun is needed to cut the Page photos: install it from https://bun.sh"))
	}
	script := filepath.Join(e.frontendDir, "tools", "cut-pages.ts")
	if _, err := os.Stat(script); err != nil {
		return fail(e.stderr, fmt.Errorf("cannot find the cutter at %s: run make tune-scan from the backend folder, which sits beside frontend/", script))
	}

	d, cleanup, err := e.open()
	if err != nil {
		return fail(e.stderr, err)
	}
	defer cleanup()
	if err := requireEmbeddings(ctx, d); err != nil {
		return fail(e.stderr, err)
	}
	// Built before cutting so a missing Gemini key fails before any work.
	embedder, err := e.newEmbedder(ctx)
	if err != nil {
		return fail(e.stderr, err)
	}

	outDir, err := cellsFolder(*cutDir)
	if err != nil {
		return fail(e.stderr, err)
	}
	if *cutDir == "" {
		defer removeFolder(outDir, e.stderr)
	}
	if err := e.cut(ctx, e.frontendDir, photosDir, outDir, g); err != nil {
		return fail(e.stderr, err)
	}
	cells, err := readCutCells(outDir)
	if err != nil {
		return fail(e.stderr, err)
	}

	scores, err := scoreCells(ctx, d, embedder, cells)
	if err != nil {
		return fail(e.stderr, err)
	}
	rep := &report{w: e.stdout}
	writeCellReport(rep, d.model, g, scores, *threshold, counting)
	if rep.err != nil {
		return fail(e.stderr, rep.err)
	}
	return 0
}

// cellsFolder is the folder the cut cells go to: the one --cut-dir names, or a
// new temporary one.
func cellsFolder(cutDir string) (string, error) {
	if cutDir != "" {
		abs, err := filepath.Abs(cutDir)
		if err != nil {
			return "", fmt.Errorf("resolving --cut-dir: %w", err)
		}
		return abs, nil
	}
	tmp, err := os.MkdirTemp("", "tune-scan-cells-")
	if err != nil {
		return "", fmt.Errorf("making a folder for the cut cells: %w", err)
	}
	return tmp, nil
}

func removeFolder(dir string, stderr io.Writer) {
	if err := os.RemoveAll(dir); err != nil {
		printf(stderr, "tune-scan: could not remove %s: %v\n", dir, err)
	}
}

// readCutCells lists the cell files under outDir: photo by photo in name order,
// and within a photo in row, then column, order.
func readCutCells(outDir string) ([]cutCell, error) {
	photoDirs, err := os.ReadDir(outDir)
	if err != nil {
		return nil, fmt.Errorf("reading the cut cells: %w", err)
	}

	var cells []cutCell
	for _, photoDir := range photoDirs {
		if !photoDir.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(outDir, photoDir.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading the cut cells of %s: %w", photoDir.Name(), err)
		}
		var photoCells []cutCell
		for _, file := range files {
			match := cellFileName.FindStringSubmatch(file.Name())
			if match == nil {
				continue
			}
			row, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, fmt.Errorf("reading the row of %s: %w", file.Name(), err)
			}
			column, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, fmt.Errorf("reading the column of %s: %w", file.Name(), err)
			}
			photoCells = append(photoCells, cutCell{
				photo:  photoDir.Name(),
				row:    row,
				column: column,
				path:   filepath.Join(outDir, photoDir.Name(), file.Name()),
			})
		}
		sort.Slice(photoCells, func(i, j int) bool {
			a, b := photoCells[i], photoCells[j]
			return a.row < b.row || (a.row == b.row && a.column < b.column)
		})
		cells = append(cells, photoCells...)
	}
	return cells, nil
}

// scoreCells embeds each cell and scores it the way the matcher does: the same
// candidate limit, and the same ScoreNearest over the nearest embeddings.
func scoreCells(ctx context.Context, d deps, embedder coreembedding.ImageEmbedder, cells []cutCell) ([]cellScore, error) {
	scores := make([]cellScore, 0, len(cells))
	for i, cell := range cells {
		jpeg, err := os.ReadFile(cell.path)
		if err != nil {
			return nil, fmt.Errorf("reading cell %s: %w", cell.path, err)
		}
		vector, err := embedder.Embed(ctx, jpegMIME, jpeg)
		if err != nil {
			return nil, fmt.Errorf("embedding %s r%02dc%02d: %w", cell.photo, cell.row, cell.column, err)
		}
		nearest, err := d.embeddings.NearestCards(ctx, d.model, vector, service.MatchCandidates)
		if err != nil {
			return nil, err
		}

		score := cellScore{cutCell: cell}
		if top := service.ScoreNearest(nearest); len(top) > 0 {
			rows, err := d.cards.CardsByIDs(ctx, []uuid.UUID{top[0].CardID})
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				return nil, fmt.Errorf("card %s of %s r%02dc%02d is gone", top[0].CardID, cell.photo, cell.row, cell.column)
			}
			score.scored = true
			score.similarity = top[0].Similarity
			score.cardName = rows[0].Name
			score.setCode = rows[0].ExpansionSetCode
		}
		logger.Infof("[%d/%d] %s r%02dc%02d scored", i+1, len(cells), cell.photo, cell.row, cell.column)
		scores = append(scores, score)
	}
	return scores, nil
}

// writeCellReport prints the cells grouped by photo, then each photo's summary.
func writeCellReport(r *report, model string, g grid, scores []cellScore, threshold float64, counting bool) {
	header := fmt.Sprintf("model %s, %dx%d grid", model, g.rows, g.columns)
	if counting {
		header += fmt.Sprintf(", threshold %.3f", threshold)
	}
	r.printf("%s\n", header)

	for start := 0; start < len(scores); {
		end := start
		for end < len(scores) && scores[end].photo == scores[start].photo {
			end++
		}
		writePhotoReport(r, scores[start:end], threshold, counting)
		start = end
	}
}

func writePhotoReport(r *report, cells []cellScore, threshold float64, counting bool) {
	r.printf("\n%s\n", cells[0].photo)

	lowest, highest := math.Inf(1), math.Inf(-1)
	scored, below := 0, 0
	for _, c := range cells {
		if !c.scored {
			r.printf("  row %d col %d  -  no published Card to compare\n", c.row, c.column)
			continue
		}
		r.printf("  row %d col %d  %.3f  %s (%s)\n", c.row, c.column, c.similarity, c.cardName, c.setCode)
		scored++
		lowest = min(lowest, c.similarity)
		highest = max(highest, c.similarity)
		if c.similarity < threshold {
			below++
		}
	}

	if scored == 0 {
		r.printf("  no cell had a published Card to compare\n")
		return
	}
	summary := fmt.Sprintf("  lowest %.3f, highest %.3f", lowest, highest)
	if counting {
		summary += fmt.Sprintf(", %d of %d below %.3f", below, scored, threshold)
	}
	r.printf("%s\n", summary)
}
