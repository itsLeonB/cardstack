import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

// Blocks shell reads and in-place edits of code files. docs/agents/conventions/serena.md requires Serena's
// symbolic tools for both, and the serena-hooks reminder only covers grep-shaped commands, so this path was open.
// Escape hatch: name SERENA_OK=1 anywhere in the command when Serena failed or the file is not parseable as code.
const READ_COMMANDS = new Set(["cat", "head", "tail", "nl", "sed", "less", "more", "bat"]);
const WRAPPERS = new Set(["sudo", "command", "timeout", "nice", "rtk"]);
const CODE_FILE = /\.(go|ts|tsx|js|jsx)$/;
const PLACE_FLAG = /^(?:-[a-zA-Z]*i|--in-place)$/;
const WRITES_CODE = />{1,2}\s*['"]?[^\s'"]*\.(go|ts|tsx|js|jsx)\b/;
const ASSIGNMENT = /^[A-Za-z_][A-Za-z0-9_]*=/;
const NUMBER = /^\d+[smhd]?$/;
const SEPARATORS = /&&|\|\||[;|]/;

const unquote = (word: string) => word.replace(/^['"]|['"]$/g, "");

function codeOperand(args: string[]): string | undefined {
  return args.map(unquote).find((word) => CODE_FILE.test(word));
}

export function decide(toolName: string, input: Record<string, unknown>): string {
  if (toolName !== "bash") return "";
  const cmd = String(input.command ?? "");
  if (cmd.includes("SERENA_OK=1")) return "";

  for (const segment of cmd.split(SEPARATORS)) {
    if (WRITES_CODE.test(segment)) continue; // redirect into a code file creates it: no symbol to edit yet
    const words = segment.trim().split(/\s+/).filter(Boolean);
    let i = 0;
    while (i < words.length && (ASSIGNMENT.test(words[i]) || WRAPPERS.has(words[i]) || NUMBER.test(words[i]))) i++;
    const command = words[i];
    if (!command || !READ_COMMANDS.has(command)) continue;
    const args = words.slice(i + 1);
    const path = codeOperand(args);
    if (!path) continue;
    const inPlace = command === "sed" && args.some((word) => PLACE_FLAG.test(word));
    return inPlace
      ? `${path} is a code file: edit it with Serena (replace_symbol_body, insert_before_symbol, insert_after_symbol, replace_content), not sed -i. Name SERENA_OK=1 in the command to override when Serena failed or the file is not parseable as code.`
      : `${path} is a code file: read it with Serena (get_symbols_overview, then find_symbol with include_body), or ctx_execute_file to extract only what you need. Name SERENA_OK=1 in the command to override when Serena failed or the file is not parseable as code.`;
  }
  return "";
}

export default function (pi: ExtensionAPI) {
  pi.on("tool_call", async (event) => {
    const reason = decide(event.toolName, event.input);
    if (reason) return { block: true, reason };
  });
}
