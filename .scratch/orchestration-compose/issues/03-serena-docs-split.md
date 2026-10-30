# 03: Serena docs split

**What to build:** an agent in any harness can find and reach Serena's tools from the agent doc alone, and a maintainer using Claude Code or pi can understand and fix the wiring from a human doc.

**Blocked by:** None (can start immediately).

**Delivery:** PR 1, branch `docs/orchestration-review-flow`, orchestrator-owned (docs).

**Status:** done

- [x] The agent doc names no harness. It tells the agent to find Serena's tools by Serena's own names whatever prefix the harness adds, and to connect or enable the Serena server through the harness's own MCP mechanism when no Serena tool is listed, falling back to built-in tools only after that fails.
- [x] The agent doc says to call the project-activation tool only if it exists. The tool mapping table and the pre-edit workflow stay, keyed by Serena's own names. The exceptions for built-in tools stay.
- [x] The human doc covers Claude Code and pi only, with no generic-harness section. It records pi's tool naming and the connect step that caused the surprise in the ticket 06 session.
- [x] Setup details that were in the agent doc for one client move to the human doc. The Serena setup itself is unchanged.
- [x] The repo's convention check passes and no markdown line is hand-wrapped.
