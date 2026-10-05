#!/usr/bin/env bash
# PreToolUse(Bash): remind, never block, when a command reads a code file with cat/head/tail/sed/nl.
# docs/agents/conventions/serena.md allows these reads in a few cases, so this only points back at the policy.
# Subagents (payload carries agent_id) use the built-in tools by policy, so they are never reminded.
input=$(cat)
[ -n "$(jq -r '.agent_id // empty' <<<"$input")" ] && exit 0
cmd=$(jq -r '.tool_input.command // ""' <<<"$input")
grep -Eq '(^|[;&|(] *)(cat|head|tail|nl|sed +-n)\b[^;&|]*\.(go|tsx?|jsx?)\b' <<<"$cmd" || exit 0
jq -n '{hookSpecificOutput: {hookEventName: "PreToolUse", additionalContext: "This Bash command reads a code file. Per docs/agents/conventions/serena.md, read code with get_symbols_overview then find_symbol (include_body=true); in Go, search a method by its bare name (Verify), not Type/Method. Continue only if one of the policy exceptions applies."}}'
