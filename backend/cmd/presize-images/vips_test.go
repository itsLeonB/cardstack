package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/itsLeonB/cardstack/backend/internal/domain/entity"
	"github.com/itsLeonB/cardstack/backend/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_MissingVipsExitsNonZeroAndTouchesNothing(t *testing.T) {
	opened := false
	open := func() (deps, func(), error) {
		opened = true
		return deps{sets: mocks.NewMockExpansionSetRepository(t), store: mocks.NewMockObjectStore(t)}, func() {}, nil
	}
	missing := func(string) (string, error) { return "", errors.New("not found") }

	code := run(context.Background(), "", missing, fakeResize, open)

	assert.NotZero(t, code)
	assert.False(t, opened, "config, database and bucket must not be touched when vips is missing")
}

func TestRequireVips_NamesTheFix(t *testing.T) {
	err := requireVips(func(string) (string, error) { return "", errors.New("not found") })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vips is not installed")
	assert.NoError(t, requireVips(func(string) (string, error) { return "/usr/bin/vips", nil }))
}

func TestRun_FailedRowExitsNonZero(t *testing.T) {
	set := newSet("A")
	sets := mocks.NewMockExpansionSetRepository(t)
	sets.EXPECT().ListMissingCovers(context.Background(), "").Return([]entity.ExpansionSet{set}, nil)
	store := mocks.NewMockObjectStore(t)
	store.EXPECT().Get(context.Background(), set.ImageKey).Return(nil, errors.New("r2 down"))
	cleaned := false
	open := func() (deps, func(), error) { return deps{sets: sets, store: store}, func() { cleaned = true }, nil }
	found := func(string) (string, error) { return "/usr/bin/vips", nil }

	assert.NotZero(t, run(context.Background(), "", found, fakeResize, open))
	assert.True(t, cleaned)
}

// pngWithAlpha encodes a w x h PNG whose left half is fully transparent.
func pngWithAlpha(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := w / 2; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 30, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func vipsHeader(t *testing.T, field, file string) int {
	t.Helper()
	out, err := exec.Command("vipsheader", "-f", field, file).Output()
	require.NoError(t, err)
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	require.NoError(t, err)
	return n
}

// TestVipsResize_LongestSideIs128 runs the real vips (skipped when it is not
// installed): portrait and landscape both end with a 128px longest side, as
// WebP, with alpha kept.
func TestVipsResize_LongestSideIs128(t *testing.T) {
	if err := requireVips(exec.LookPath); err != nil {
		t.Skip("vips not installed")
	}
	if _, err := exec.LookPath("vipsheader"); err != nil {
		t.Skip("vipsheader not installed")
	}

	tests := []struct {
		name               string
		w, h, wantW, wantH int
	}{
		{"portrait", 400, 600, 85, 128},
		{"landscape", 600, 400, 128, 85},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cover, err := vipsResize(context.Background(), pngWithAlpha(t, tt.w, tt.h))
			require.NoError(t, err)
			require.True(t, bytes.HasPrefix(cover, []byte("RIFF")) && bytes.Equal(cover[8:12], []byte("WEBP")), "output is WebP")

			file := filepath.Join(t.TempDir(), "cover.webp")
			require.NoError(t, os.WriteFile(file, cover, 0o600))
			assert.Equal(t, tt.wantW, vipsHeader(t, "width", file))
			assert.Equal(t, tt.wantH, vipsHeader(t, "height", file))
			assert.Equal(t, 4, vipsHeader(t, "bands", file), "alpha is kept")
		})
	}
}
