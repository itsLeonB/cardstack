package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecide(t *testing.T) {
	dir := t.TempDir()
	big, small := filepath.Join(dir, "big.txt"), filepath.Join(dir, "small.txt")
	os.WriteFile(small, []byte("hi\n"), 0o644)
	os.WriteFile(big, []byte("x"), 0o644)
	run := func(args ...string) {
		c := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=a@b", "-c", "user.name=t"}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	run("init", "-q")
	run("add", "-A")
	run("commit", "-qm", "init")
	os.WriteFile(big, []byte(strings.Repeat("y", 50000)), 0o644)
	t.Chdir(dir)

	bash := func(c string) input { return input{"Bash", map[string]any{"command": c}} }
	read := func(m map[string]any) input { return input{"Read", m} }
	cases := []struct {
		name string
		in   input
		deny bool
	}{
		{"cat big", bash("cat " + big), true},
		{"cat small", bash("cat " + small), false},
		{"cat big piped", bash("cat " + big + " | head -5"), false},
		{"cat big BIG_OK", bash("BIG_OK=1 cat " + big), false},
		{"git diff big", bash("git diff"), true},
		{"git diff --stat", bash("git diff --stat"), false},
		{"git diff small path", bash("git diff -- small.txt"), false},
		{"read big no limit", read(map[string]any{"file_path": big}), true},
		{"read big limit", read(map[string]any{"file_path": big, "limit": 100.0}), false},
		{"read small", read(map[string]any{"file_path": small}), false},
		{"other tool", input{"Edit", map[string]any{"file_path": "x"}}, false},
	}
	for _, c := range cases {
		if got := decide(c.in) != ""; got != c.deny {
			t.Errorf("%s: deny=%v want %v", c.name, got, c.deny)
		}
	}
}
