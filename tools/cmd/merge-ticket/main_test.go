package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repo builds a throwaway repo on feat/int with a backend/ dir and an issue file; commitIn adds a worktree on a new branch with one committed file.
func repo(t *testing.T) (main string, g func(dir string, args ...string) string, commitIn func(branch, file string) string) {
	tmp := t.TempDir()
	main = filepath.Join(tmp, "main")
	os.Mkdir(main, 0o755)
	g = func(dir string, args ...string) string {
		t.Helper()
		out, err := git(append([]string{"-C", dir, "-c", "user.email=a@b", "-c", "user.name=t"}, args...)...)
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
		return out
	}
	commitIn = func(branch, file string) string {
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
	return
}

func TestMergeTicket(t *testing.T) {
	main, g, commitIn := repo(t)
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

func TestScopeValues(t *testing.T) {
	main, _, commitIn := repo(t)
	commitIn("feat/int-multi", "backend/b")
	commitIn("feat/int-multi2", "docs/c")
	commitIn("feat/int-repeat", "docs/d")
	t.Chdir(main)

	if err := run([]string{"merge", "feat/int-multi", "--scope", "backend", "docs"}); err != nil {
		t.Fatalf("several values after one flag: %v", err)
	}
	if err := run([]string{"merge", "feat/int-multi2", "--scope", "backend", "--scope", "docs/"}); err != nil {
		t.Fatalf("repeated flag: %v", err)
	}
	if err := run([]string{"merge", "feat/int-repeat", "--scope"}); err == nil {
		t.Fatal("--scope without a value should be refused")
	}
}

func TestComponentExpansion(t *testing.T) {
	main, _, commitIn := repo(t)
	commitIn("feat/int-fe", "admin/x")
	commitIn("feat/int-fe2", "frontend/y")
	commitIn("feat/int-wrong", "backend/z")
	t.Chdir(main)

	if err := run([]string{"merge", "feat/int-wrong", "--component", "frontend"}); err == nil || !strings.Contains(err.Error(), "backend/z") {
		t.Fatalf("frontend component must not cover backend/, got %v", err)
	}
	if err := run([]string{"merge", "feat/int-fe", "--component", "frontend"}); err != nil {
		t.Fatalf("frontend covers admin/: %v", err)
	}
	if err := run([]string{"merge", "feat/int-fe2", "--component", "frontend"}); err != nil {
		t.Fatalf("frontend covers frontend/: %v", err)
	}
	if err := run([]string{"merge", "feat/int-wrong", "--component", "nope"}); err == nil || !strings.Contains(err.Error(), "unknown component") {
		t.Fatalf("unknown component should be refused, got %v", err)
	}
}

func TestWorktreeForm(t *testing.T) {
	main, g, commitIn := repo(t)
	wtRen := commitIn("feat/int-orig", "backend/b")
	g(wtRen, "branch", "-m", "harness/renamed")
	wtDet := commitIn("feat/int-det", "backend/c")
	g(wtDet, "checkout", "-q", "--detach")
	// a repository of its own: a branch name read from it means nothing in main
	foreign := filepath.Join(t.TempDir(), "foreign")
	os.Mkdir(foreign, 0o755)
	g(foreign, "init", "-q", "-b", "feat/int-orig")
	g(foreign, "commit", "-q", "--allow-empty", "-m", "x")
	t.Chdir(main)

	for _, p := range []string{foreign, filepath.Join(foreign, "nope")} {
		if err := run([]string{"merge", "--worktree", p, "--component", "backend"}); err == nil || !strings.Contains(err.Error(), "not a worktree of this repository") {
			t.Fatalf("%s should be refused as not attached, got %v", p, err)
		}
	}
	if err := run([]string{"cleanup", "--worktree", foreign}); err == nil || !strings.Contains(err.Error(), "not a worktree of this repository") {
		t.Fatalf("cleanup of a foreign repo should be refused, got %v", err)
	}
	if err := run([]string{"merge", "--worktree", wtDet, "--component", "backend"}); err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("detached HEAD should be refused, got %v", err)
	}
	if err := run([]string{"merge", "--worktree", wtRen, "--component", "backend"}); err != nil {
		t.Fatalf("worktree form after rename: %v", err)
	}
	if err := run([]string{"cleanup", "--worktree", wtRen}); err != nil {
		t.Fatalf("cleanup by worktree: %v", err)
	}
	if _, err := os.Stat(wtRen); err == nil {
		t.Error("worktree not removed")
	}
	if _, err := git("rev-parse", "-q", "--verify", "refs/heads/harness/renamed"); err == nil {
		t.Error("renamed branch not deleted")
	}
}
