# 13: Serena reminder for shell reads, not a hard deny

**What to build:** The `@lystran/pi-serena-hooks` reminder should cover read-shaped shell commands, so an agent that runs `sed -n` on a code file gets Serena's `remind` context (the nudge it already gets for `grep`) instead of the hard deny from `.pi/extensions/deny-shell-code-read.ts`.

**Blocked by:** None.

**Status:** open

The seam is `src/tool-matcher.ts` in the plugin: `normalizeSerenaRemindToolCall` maps `grep`, `ffgrep`, `multi_grep`, `fff-multi-grep` and Bash commands whose first word is in `SHELL_GREP_COMMAND_NAMES` (`ack`, `ag`, `egrep`, `fgrep`, `grep`, `rg`) to `{ toolName: "grep" }`. Adding `sed`, `cat`, `head`, `tail`, `nl` to that set is wrong: it would report a read to Serena as a search and corrupt the state `remind` compares against. The ticket is to find the read-shaped equivalent.

Open questions to settle before coding:

1. Which `toolName` does Serena's `--client claude-code` hook accept for a read (`Read`? `read_file`?), and does `remind` treat it as "you already read this symbol" rather than "you already searched"?
2. Should the normalization be word-based (first word only, as today) or should it also see through `cd backend &&` and `timeout 60`, which the current matcher does not?
3. Does the native `read` tool need the same treatment? `deny-shell-code-read.ts` covers bash only, so a whole-file `read` of a code file is still unguarded; that is deliberate until this ticket answers question 1.

## Comments

2026-11-01, from the Claude Code retro (`.scratch/agent-tooling/` history in `~/.claude/projects`): 433 `sed -n` calls on `.go`/`.ts`/`.tsx` files and 227 built-in reads of code files went out without any reminder, against 3,669 Serena calls. The hard-deny guard shipped as the cheap half; this ticket is the better half.
