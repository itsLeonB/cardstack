// Command merge-ticket runs the mechanical steps of docs/agents/orchestration.md "Merge and clean up" for one child branch.
// Run it from the feature-branch checkout.
//
//	merge-ticket merge   <branch> --scope <dir>...   clean-worktree check, scope check (step 7), then merge --no-ff (step 9.1)
//	merge-ticket cleanup <branch> [--issue <file>]   worktree remove, safe branch delete (steps 9.3-9.4), mark the issue done
//
// Archive the T3 thread (step 9.2) between the two. Install (from tools/): go install ./cmd/...
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// protected mirrors PROTECTED in .claude/hooks/block-commit-on-main.sh.
var protected = []string{"main"}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func worktreeOf(branch string) string {
	out, _ := git("worktree", "list", "--porcelain")
	var path string
	for _, l := range strings.Split(out, "\n") {
		if p, ok := strings.CutPrefix(l, "worktree "); ok {
			path = p
		}
		if l == "branch refs/heads/"+branch {
			return path
		}
	}
	return ""
}

var (
	status = regexp.MustCompile(`(?m)^\*\*Status:\*\* ready-for-agent`)
	box    = regexp.MustCompile(`(?m)^- \[ \]`)
)

type opts struct {
	branch, issue string
	scopes        []string
}

func parse(args []string) (opts, error) {
	if len(args) == 0 {
		return opts{}, errors.New("missing <branch>")
	}
	o := opts{branch: args[0]}
	for i := 1; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return o, fmt.Errorf("%s needs a value", args[i])
		}
		switch args[i] {
		case "--scope":
			o.scopes = append(o.scopes, strings.TrimSuffix(args[i+1], "/"))
		case "--issue":
			o.issue = args[i+1]
		default:
			return o, fmt.Errorf("unknown arg %s", args[i])
		}
	}
	return o, nil
}

func merge(o opts) error {
	if len(o.scopes) == 0 {
		return errors.New("--scope <dir> is required: the scope check is not optional")
	}
	target, err := git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("not on a branch: %s", target)
	}
	for _, p := range protected {
		if target == p {
			return fmt.Errorf("%s is protected: switch to the feature branch first", target)
		}
	}
	if out, _ := git("status", "--porcelain"); out != "" {
		return errors.New("working tree dirty in the feature checkout")
	}
	if _, err := git("rev-parse", "--verify", "-q", "refs/heads/"+o.branch); err != nil {
		return fmt.Errorf("no branch %s", o.branch)
	}
	if wt := worktreeOf(o.branch); wt != "" {
		if out, _ := git("-C", wt, "status", "--porcelain"); out != "" {
			return fmt.Errorf("worktree %s has uncommitted changes (invisible to the scope diff); send the child back to commit or discard:\n%s", wt, out)
		}
	}
	files, _ := git("diff", "--name-only", target+"..."+o.branch)
	var outside []string
	for _, f := range strings.Fields(files) {
		in := false
		for _, s := range o.scopes {
			in = in || strings.HasPrefix(f, s+"/")
		}
		if !in {
			outside = append(outside, f)
		}
	}
	if len(outside) > 0 {
		return fmt.Errorf("%s touches files outside %s:\n%s", o.branch, strings.Join(o.scopes, ", "), strings.Join(outside, "\n"))
	}
	if out, err := git("merge", "--no-ff", o.branch, "-m", fmt.Sprintf("Merge branch '%s' into %s", o.branch, target)); err != nil {
		git("merge", "--abort")
		return fmt.Errorf("merge failed, aborted:\n%s", out)
	}
	last, _ := git("log", "--oneline", "-1")
	fmt.Printf("merged %s into %s: %s\n", o.branch, target, last)
	return nil
}

func cleanup(o opts) error {
	if wt := worktreeOf(o.branch); wt != "" {
		if out, err := git("worktree", "remove", wt); err != nil { // never --force: leftover state goes to the maintainer
			return fmt.Errorf("worktree remove refused, report to the maintainer instead of forcing: %s", out)
		}
		fmt.Println("removed worktree", wt)
	}
	if out, err := git("branch", "-d", o.branch); err != nil { // never -D
		return fmt.Errorf("branch -d refused (unmerged?): %s", out)
	}
	fmt.Println("deleted branch", o.branch)
	if o.issue != "" {
		b, err := os.ReadFile(o.issue)
		if err != nil {
			return err
		}
		b = box.ReplaceAll(status.ReplaceAll(b, []byte("**Status:** done")), []byte("- [x]"))
		if err := os.WriteFile(o.issue, b, 0o644); err != nil {
			return err
		}
		fmt.Println("marked done:", o.issue)
	}
	return nil
}

func run(args []string) error {
	if len(args) == 0 || (args[0] != "merge" && args[0] != "cleanup") {
		return errors.New("usage: merge-ticket merge <branch> --scope <dir>... | merge-ticket cleanup <branch> [--issue <file>]")
	}
	o, err := parse(args[1:])
	if err != nil {
		return err
	}
	if args[0] == "merge" {
		return merge(o)
	}
	return cleanup(o)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ABORT:", err)
		os.Exit(1)
	}
}
