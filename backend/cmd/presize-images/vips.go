package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/itsLeonB/ungerr"
)

// coverSize is the longest side of a cover. The tile shows it at 64 CSS px, so
// 128 covers 2x displays (docs/adr/0017).
const coverSize = "128"

const vipsTimeout = time.Minute

// requireVips returns an error naming the fix when the vips binary is not on
// PATH. lookPath is exec.LookPath in production.
func requireVips(lookPath func(string) (string, error)) error {
	if _, err := lookPath("vips"); err != nil {
		return ungerr.Wrap(err, "vips is not installed or not on PATH; install libvips (e.g. `apt install libvips-tools` or `brew install vips`) and re-run")
	}
	return nil
}

// vipsResize shrinks original into a WebP whose longest side is at most
// coverSize, keeping alpha (no flatten), the same saver scripts/build-series-logos.sh
// uses. `--size down` never enlarges a smaller original.
func vipsResize(ctx context.Context, original []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "presize-images-")
	if err != nil {
		return nil, ungerr.Wrap(err, "creating temp dir")
	}
	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup of a temp dir

	in, out := filepath.Join(dir, "original"), filepath.Join(dir, "cover.webp")
	if err := os.WriteFile(in, original, 0o600); err != nil {
		return nil, ungerr.Wrap(err, "writing original to temp dir")
	}

	ctx, cancel := context.WithTimeout(ctx, vipsTimeout)
	defer cancel()
	// A box of coverSize x coverSize with the aspect ratio kept makes the
	// longest side the binding one, for portrait and landscape alike.
	cmd := exec.CommandContext(ctx, "vips", "thumbnail", in, out+"[Q=90]", coverSize, "--height", coverSize, "--size", "down")
	if combined, err := cmd.CombinedOutput(); err != nil {
		return nil, ungerr.Wrapf(err, "vips thumbnail: %s", combined)
	}

	cover, err := os.ReadFile(out)
	if err != nil {
		return nil, ungerr.Wrap(err, "reading resized cover")
	}
	return cover, nil
}
