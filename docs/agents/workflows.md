# Workflows

How the maintainer's slash commands run in this repo. Each file holds the repo rules around the generic skill of the same name.

| Invocation | File |
| --- | --- |
| `/implement` | `workflows/implement.md` |
| `/implement-spec` | `workflows/implement-spec.md` |

Commit and branch naming for every workflow: [`docs/git.md`](../git.md).

A role file under `docs/agents/roles/` points at reference docs (conventions, `handoff.md`, `git.md`, an ADR) and never at a workflow file: the workflow names the role, not the reverse.

## Codify repeats

When the same multi-step pattern recurs (after a retro, or when the same instructions are retyped in prompts), propose a workflow to the maintainer: a new file in `workflows/` and a row in the table above. Each doc adds to what the maintainer must remember, so create it only after they agree. The skills `implement`, `implement-spec`, `code-review`, `tdd` and `retro` are vendored: never edit them, put the repo rule in a workflow file.
