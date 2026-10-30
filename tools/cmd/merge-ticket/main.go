// Command merge-ticket runs the mechanical steps of docs/agents/orchestration.md "Merge and clean up" for one child branch.
// Run it from the feature-branch checkout.
//
//	merge-ticket merge   <branch> --scope <dir>...      clean-worktree check, scope check (step 6), then merge --no-ff (step 7.1)
//	merge-ticket merge   --worktree <path> --component <name>...   the same, with the branch read from the worktree's live HEAD and the scope from the component map
//	merge-ticket cleanup <branch> [--issue <file>]      worktree remove, safe branch delete (steps 7.2-7.3), mark the issue done
//	merge-ticket cleanup --worktree <path> [--issue <file>]
//
// --scope and --component take several values (or repeat); give <branch> before them. A detached HEAD in the worktree is refused.
//
// Install (from tools/): go install ./cmd/...
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
	branch, issue, worktree string
	scopes, components      []string
}

// components is the only map from a component name to its directories (ADR-0024: the frontend component covers two packages).
var components = map[string][]string{
	"backend":  {"backend"},
	"frontend": {"frontend", "admin"},
}

// resolveBranch fills o.branch from the live branch of o.worktree, so a renamed or harness-minted branch needs no copying by hand.
func resolveBranch(o *opts) error {
	if o.worktree == "" {
		return nil
	}
	out, err := git("-C", o.worktree, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("%s has a detached HEAD or is not a worktree, refusing: %s", o.worktree, out)
	}
	o.branch = out
	return nil
}

// expandComponents adds the directories of each --component to o.scopes.
func expandComponents(o *opts) error {
	for _, c := range o.components {
		dirs, ok := components[c]
		if !ok {
			return fmt.Errorf("unknown component %s", c)
		}
		o.scopes = append(o.scopes, dirs...)
	}
	return nil
}

func parse(args []string) (opts, error) {
	var o opts
	var multi *[]string // the list a bare value currently extends, nil outside --scope/--component
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			switch {
			case multi != nil:
				*multi = append(*multi, strings.TrimSuffix(a, "/"))
			case o.branch == "":
				o.branch = a
			default:
				return o, fmt.Errorf("unexpected arg %s", a)
			}
			continue
		}
		multi = nil
		switch a {
		case "--scope":
			multi = &o.scopes
		case "--component":
			multi = &o.components
		case "--issue", "--worktree":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s needs a value", a)
			}
			i++
			if a == "--issue" {
				o.issue = args[i]
			} else {
				o.worktree = args[i]
			}
		default:
			return o, fmt.Errorf("unknown arg %s", a)
		}
		if multi != nil && (i+1 >= len(args) || strings.HasPrefix(args[i+1], "--")) {
			return o, fmt.Errorf("%s needs a value", a)
		}
	}
	switch {
	case o.branch != "" && o.worktree != "":
		return o, errors.New("give <branch> or --worktree, not both")
	case o.branch == "" && o.worktree == "":
		return o, errors.New("missing <branch> or --worktree <path>")
	}
	return o, nil
}

func merge(o opts) error {
	if len(o.scopes) == 0 {
		return errors.New("--scope <dir> or --component <name> is required: the scope check is not optional")
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
		return errors.New("usage: merge-ticket merge (<branch> | --worktree <path>) (--component <name>... | --scope <dir>...) | merge-ticket cleanup (<branch> | --worktree <path>) [--issue <file>]")
	}
	o, err := parse(args[1:])
	if err != nil {
		return err
	}
	if err := resolveBranch(&o); err != nil {
		return err
	}
	if args[0] == "merge" {
		if err := expandComponents(&o); err != nil {
			return err
		}
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
