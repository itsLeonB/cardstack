// Command deny-big-output is a PreToolUse hook (matcher Bash|Read) that denies calls that would
// dump large output into context. Telemetry (Oct 2026): unfiltered cat / git diff / limit-less Read
// were ~15% of tool output.
//
// Install (from tools/): go install ./cmd/...   (puts it on PATH so every worktree finds it)
// Escape hatch: prefix a Bash command with "BIG_OK=1 "; for Read, pass offset or limit.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type input struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

var (
	gitDiff = regexp.MustCompile(`^git (-C \S+ )?diff($| )`)
	shaped  = regexp.MustCompile(`--stat|--shortstat|--numstat|--name-only|--name-status|--quiet|-q`)
)

func hasMeta(s string) bool { return strings.ContainsAny(s, "|;&><$`") }

func limit(env string, def int64) int64 {
	if n, err := strconv.ParseInt(os.Getenv(env), 10, 64); err == nil {
		return n
	}
	return def
}

func size(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

// decide returns a non-empty reason when the call should be denied.
func decide(in input) string {
	bashMax := limit("DENY_BIG_BASH_MAX", 20000)
	readMax := limit("DENY_BIG_READ_MAX", 40000)

	switch in.ToolName {
	case "Read":
		if _, ok := in.ToolInput["offset"]; ok {
			return ""
		}
		if _, ok := in.ToolInput["limit"]; ok {
			return ""
		}
		f, _ := in.ToolInput["file_path"].(string)
		if n := size(f); n > readMax {
			return fmt.Sprintf("%s is %d bytes. Re-run Read with offset/limit, or use ctx_execute_file to extract only what you need.", f, n)
		}
	case "Bash":
		cmd, _ := in.ToolInput["command"].(string)
		if strings.HasPrefix(cmd, "BIG_OK=1 ") || hasMeta(cmd) { // piped/chained/redirected: caller already shapes output
			return ""
		}
		words := strings.Fields(cmd)
		switch {
		case len(words) >= 2 && words[0] == "cat":
			var total int64
			for _, f := range words[1:] {
				if !strings.HasPrefix(f, "-") {
					total += size(f)
				}
			}
			if total > bashMax {
				return fmt.Sprintf("cat would print %d bytes. Use Read with offset/limit, or ctx_execute_file to filter. Prefix BIG_OK=1 to override.", total)
			}
		case gitDiff.MatchString(cmd) && !shaped.MatchString(cmd):
			out, _ := exec.Command(words[0], words[1:]...).Output()
			if n := len(out); int64(n) > bashMax {
				return fmt.Sprintf("git diff would print %d bytes. Run it with --stat first, diff one path at a time, or use ctx_execute to filter. Prefix BIG_OK=1 to override.", n)
			}
		}
	}
	return ""
}

func main() {
	var in input
	if json.NewDecoder(os.Stdin).Decode(&in) != nil {
		return // malformed input: never block the tool call
	}
	if reason := decide(in); reason != "" {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"hookSpecificOutput": map[string]any{
			"hookEventName": "PreToolUse", "permissionDecision": "deny", "permissionDecisionReason": reason}})
	}
}
