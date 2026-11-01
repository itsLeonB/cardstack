import { execFileSync } from "node:child_process";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// Blocks the call up front instead of running the command and reporting the failure after.
const COMMIT_OR_MERGE = /(^|\s)git(\s+-C\s+\S+)?\s+(commit|merge)(?=\s|$)/;

export default function (pi: ExtensionAPI) {
  pi.on("tool_call", async (event, ctx) => {
    if (event.toolName !== "bash") return;
    if (!COMMIT_OR_MERGE.test(String(event.input.command ?? ""))) return;

    let branch: string;
    try {
      branch = execFileSync("git", ["-C", ctx.cwd, "branch", "--show-current"], { encoding: "utf8" }).trim();
    } catch {
      return; // not a git repo, or git unavailable: do not block
    }
    if (branch !== "main") return;

    return {
      block: true,
      reason:
        "Never commit or merge on main. Create a feature branch first: git switch -c <type>/<name> (docs/git.md).",
    };
  });
}
