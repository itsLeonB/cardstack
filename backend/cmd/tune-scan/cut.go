package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

// grid is a Page's rows and columns, as cut-pages reads them.
type grid struct {
	rows    int
	columns int
}

var gridText = regexp.MustCompile(`^(\d+)x(\d+)$`)

func parseGrid(text string) (grid, error) {
	match := gridText.FindStringSubmatch(text)
	if match == nil {
		return grid{}, fmt.Errorf("--grid must be rows x columns, like 3x3")
	}
	rows, errRows := strconv.Atoi(match[1])
	columns, errColumns := strconv.Atoi(match[2])
	if errRows != nil || errColumns != nil || rows < 1 || columns < 1 {
		return grid{}, fmt.Errorf("--grid must be rows x columns, like 3x3")
	}
	return grid{rows: rows, columns: columns}, nil
}

// cutter writes the cells of every photo in photosDir under outDir, as
// <outDir>/<photo>/r<RR>c<CC>.jpg. The real one is cutWithBun.
type cutter func(ctx context.Context, frontendDir, photosDir, outDir string, g grid) error

// cutWithBun runs frontend/tools/cut-pages.ts with bun, from the frontend folder.
// Its own progress lines go to stderr, so stdout stays the report.
func cutWithBun(ctx context.Context, frontendDir, photosDir, outDir string, g grid) error {
	cmd := exec.CommandContext(ctx, "bun", "tools/cut-pages.ts", photosDir, outDir, "--grid", fmt.Sprintf("%dx%d", g.rows, g.columns))
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cutting the Page photos with frontend/tools/cut-pages.ts: %w", err)
	}
	return nil
}
