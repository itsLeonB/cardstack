import { execFileSync } from "node:child_process";
import { statSync } from "node:fs";
import { resolve } from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// Blocks calls that would dump large output into context.
// Escape hatch: prefix a bash command with "BIG_OK=1 "; for read, pass offset or limit.
const GIT_DIFF = /^git (-C \S+ )?diff($| )/;
const SHAPED = /--stat|--shortstat|--numstat|--name-only|--name-status|--quiet|-q/;
const HAS_META = /[|;&><$`]/;

const limit = (env: string, def: number) => {
  const n = Number.parseInt(process.env[env] ?? "", 10);
  return Number.isNaN(n) ? def : n;
};

const size = (path: string) => {
  try {
    return statSync(path).size;
  } catch {
    return 0;
  }
};

/** Returns a reason when the call should be denied, otherwise "". */
export function decide(toolName: string, input: Record<string, unknown>, cwd: string): string {
  const bashMax = limit("DENY_BIG_BASH_MAX", 20000);
  const readMax = limit("DENY_BIG_READ_MAX", 40000);

  if (toolName === "read") {
    if (input.offset !== undefined || input.limit !== undefined) return "";
    const path = String(input.path ?? "");
    const n = size(resolve(cwd, path));
    if (n > readMax) {
      return `${path} is ${n} bytes. Re-run read with offset/limit, or use ctx_execute_file to extract only what you need.`;
    }
  } else if (toolName === "bash") {
    const cmd = String(input.command ?? "");
    if (cmd.startsWith("BIG_OK=1 ") || HAS_META.test(cmd)) return ""; // piped/chained/redirected: caller already shapes output
    const words = cmd.split(/\s+/).filter(Boolean);
    if (words[0] === "cat" && words.length >= 2) {
      const total = words.slice(1).filter((f) => !f.startsWith("-")).reduce((sum, f) => sum + size(resolve(cwd, f)), 0);
      if (total > bashMax) {
        return `cat would print ${total} bytes. Use read with offset/limit, or ctx_execute_file to filter. Prefix BIG_OK=1 to override.`;
      }
    } else if (GIT_DIFF.test(cmd) && !SHAPED.test(cmd)) {
      let out = "";
      try {
        out = execFileSync(words[0], words.slice(1), { cwd, encoding: "utf8", maxBuffer: 256 * 1024 * 1024, stdio: ["ignore", "pipe", "ignore"] });
      } catch {
        return ""; // git failed: let the real call report it
      }
      if (out.length > bashMax) {
        return `git diff would print ${out.length} bytes. Run it with --stat first, diff one path at a time, or use ctx_execute to filter. Prefix BIG_OK=1 to override.`;
      }
    }
  }
  return "";
}

export default function (pi: ExtensionAPI) {
  pi.on("tool_call", async (event, ctx) => {
    const reason = decide(event.toolName, event.input, ctx.cwd);
    if (reason) return { block: true, reason };
  });
}
