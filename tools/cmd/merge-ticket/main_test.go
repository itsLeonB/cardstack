package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeTicket(t *testing.T) {
	tmp := t.TempDir()
	main := filepath.Join(tmp, "main")
	os.Mkdir(main, 0o755)
	g := func(dir string, args ...string) string {
		t.Helper()
		out, err := git(append([]string{"-C", dir, "-c", "user.email=a@b", "-c", "user.name=t"}, args...)...)
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return out
	}
	commitIn := func(branch, file string) string {
		wt := filepath.Join(tmp, "wt-"+filepath.Base(branch))
		g(main, "worktree", "add", "-q", "-b", branch, wt)
		os.MkdirAll(filepath.Dir(filepath.Join(wt, file)), 0o755)
		os.WriteFile(filepath.Join(wt, file), []byte("x"), 0o644)
		g(wt, "add", "-A")
		g(wt, "commit", "-qm", branch)
		return wt
	}

	g(main, "init", "-q", "-b", "feat/int")
	protected = []string{"main"}
	os.Mkdir(filepath.Join(main, "backend"), 0o755)
	os.WriteFile(filepath.Join(main, "backend", "a"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(main, "issue.md"), []byte("**Status:** ready-for-agent\n- [ ] do\n"), 0o644)
	g(main, "add", "-A")
	g(main, "commit", "-qm", "init")
	wtBe := commitIn("feat/int-be", "backend/b")
	commitIn("feat/int-bad", "docs/c")
	t.Chdir(main)

	if err := run([]string{"merge", "feat/int-be"}); err == nil {
		t.Fatal("merge without --scope should be refused")
	}
	if err := run([]string{"merge", "feat/int-bad", "--scope", "backend"}); err == nil || !strings.Contains(err.Error(), "docs/c") {
		t.Fatalf("out-of-scope should be refused naming docs/c, got %v", err)
	}
	os.WriteFile(filepath.Join(wtBe, "stray"), []byte("x"), 0o644)
	if err := run([]string{"merge", "feat/int-be", "--scope", "backend"}); err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("dirty child worktree should be refused, got %v", err)
	}
	os.Remove(filepath.Join(wtBe, "stray"))
	if err := run([]string{"merge", "feat/int-be", "--scope", "backend"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"cleanup", "feat/int-be", "--issue", "issue.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wtBe); err == nil {
		t.Error("worktree not removed")
	}
	if _, err := git("rev-parse", "-q", "--verify", "refs/heads/feat/int-be"); err == nil {
		t.Error("branch not deleted")
	}
	if b, _ := os.ReadFile("issue.md"); string(b) != "**Status:** done\n- [x] do\n" {
		t.Errorf("issue not marked: %q", b)
	}
	if n := g(main, "rev-list", "--merges", "--count", "HEAD"); n != "1" {
		t.Errorf("want one --no-ff merge commit, got %s", n)
	}
	g(main, "switch", "-q", "-c", "main")
	if err := run([]string{"merge", "feat/int-bad", "--scope", "docs"}); err == nil || !strings.Contains(err.Error(), "protected") {
		t.Errorf("merge on protected branch should be refused, got %v", err)
	}
}
