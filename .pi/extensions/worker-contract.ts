import { existsSync } from "node:fs";
import { resolve } from "node:path";
import type {
  ExtensionAPI,
  ExtensionContext,
} from "@earendil-works/pi-coding-agent";

const WORKER =
  /^Act as the (implementation|research|review|design|test|general) sub-agent for this task\./;
const SPAWN =
  /(?:^|__)(delegate_task|t3_thread_launch|create_threads|subagent)$/;

function text(content: unknown): string {
  if (typeof content === "string") return content;
  if (!Array.isArray(content)) return "";
  return content
    .filter((part) => part.type === "text")
    .map((part) => part.text)
    .join("\n");
}

export default function (pi: ExtensionAPI) {
  let task = "";
  let baseline = "";
  let reminder = false;
  const failures = new Set<string>();
  const starts = new Set<string>();
  const children = new Map<string, string>();
  const inspect = new Set<string>();

  function restore(ctx: ExtensionContext) {
    const first = ctx.sessionManager
      .getBranch()
      .find(
        (entry) => entry.type === "message" && entry.message.role === "user",
      );
    task =
      first?.type === "message" && first.message.role === "user"
        ? text(first.message.content)
        : "";
  }

  pi.on("session_start", (_event, ctx) => {
    restore(ctx);
    baseline = "";
    reminder = false;
    failures.clear();
    starts.clear();
    children.clear();
    inspect.clear();
  });
  pi.on("session_tree", (_event, ctx) => restore(ctx));

  async function snapshot(ctx: ExtensionContext) {
    const result = await pi.exec("git", ["diff", "HEAD", "--"], {
      cwd: ctx.cwd,
      timeout: 5000,
    });
    const head = await pi.exec("git", ["rev-parse", "HEAD"], {
      cwd: ctx.cwd,
      timeout: 5000,
    });
    const status = await pi.exec("git", ["status", "--porcelain"], {
      cwd: ctx.cwd,
      timeout: 5000,
    });
    return result.code === 0 && head.code === 0 && status.code === 0
      ? head.stdout + result.stdout + status.stdout
      : "";
  }

  pi.on("before_agent_start", async (event, ctx) => {
    restore(ctx);
    if (!task && event.messageType === "user") task = event.prompt;
    if (!WORKER.test(task)) return;
    baseline = await snapshot(ctx);
    reminder = false;
    failures.clear();
    const tools = pi.getAllTools().map((tool) => tool.name);
    const serena = tools.filter((name) =>
      /serena.*(?:initial_instructions|get_symbols_overview|find_symbol)/.test(
        name,
      ),
    );
    const paths = [
      ...new Set(task.match(/(?:\.scratch|docs)\/[\w./-]+\.md/g) ?? []),
    ];
    return {
      message: {
        customType: "worker-preflight",
        display: false,
        content: [
          `Worker preflight: checkout ${ctx.cwd}.`,
          ...paths.map(
            (path) =>
              `${existsSync(resolve(ctx.cwd, path)) ? "Found" : "Missing"}: ${path}`,
          ),
          `Registered Serena tools: ${serena.join(", ") || "none yet; discover/connect before declaring them unavailable"}.`,
          "Use the task above as the complete brief. Work in this checkout. If Serena discovery/connection fails, use the documented absolute-path fallback and record the failed operation. A tool absent from the initial list is not proof of failure.",
          "Finish with Outcome: implemented, verified-unchanged, blocked, or failed, followed by Evidence: describing changes, checks and any exact blocker. For blocked tool access, name the failed tool call. Preserve existing changes and distinguish your work from them.",
        ].join("\n"),
      },
    };
  });

  pi.on("tool_call", (event, ctx) => {
    restore(ctx);
    const input = event.input as { tool?: string; role?: string };
    const name = input.tool ?? event.toolName;
    if (/(?:^|__)delegate_task$/.test(name)) {
      if (inspect.size > 0)
        return {
          block: true,
          reason:
            "An implementation worker returned without an accepted outcome. Read its activity with t3_thread_read before launching another worker.",
        };
      if (input.role === "implementation") starts.add(event.toolCallId);
    }
    if (!WORKER.test(task) || /^Child delegation: allowed\s*$/m.test(task))
      return;
    if (SPAWN.test(name))
      return {
        block: true,
        reason:
          "This session is a worker. Carry out the supplied task yourself. Only an explicit 'Child delegation: allowed' line in the original task permits children. Your parent already has the coordination role.",
      };
  });

  pi.on("tool_result", (event) => {
    let data = event.structuredContent as
      | {
          taskId?: string;
          childThreadId?: string;
          status?: string;
          summary?: string;
        }
      | undefined;
    if (!data) {
      try {
        data = JSON.parse(text(event.content));
      } catch {
        return;
      }
    }
    if (!data || event.isError) return;
    if (starts.delete(event.toolCallId) && data.taskId && data.childThreadId)
      children.set(data.taskId, data.childThreadId);
    if (
      data.taskId &&
      children.has(data.taskId) &&
      ["completed", "failed", "cancelled"].includes(data.status ?? "")
    ) {
      if (
        !/^Outcome:\s*(implemented|verified-unchanged)\s*$/m.test(
          data.summary ?? "",
        ) ||
        !/^Evidence:\s*\S/m.test(data.summary ?? "")
      ) {
        inspect.add(children.get(data.taskId)!);
        return {
          structuredContent: event.structuredContent,
          content: [
            ...event.content,
            {
              type: "text" as const,
              text: "Run finished, but implementation acceptance is not established. Read this child's activity before any retry; do not lengthen and resend the same brief.",
            },
          ],
        };
      }
    }
    const input = event.input as { threadId?: string; view?: string };
    if (
      /(?:^|__)t3_thread_read$/.test(event.toolName) &&
      input.view === "activity" &&
      input.threadId
    )
      inspect.delete(input.threadId);
  });

  pi.on("tool_execution_end", (event) => {
    if (event.isError) failures.add(event.toolName);
  });

  pi.on("agent_before_settle", async (event, ctx) => {
    if (!WORKER.test(task) || !/^Act as the implementation/.test(task)) return;
    const last = [...ctx.sessionManager.getBranch()]
      .reverse()
      .find(
        (entry) =>
          entry.type === "message" && entry.message.role === "assistant",
      );
    const report =
      last?.type === "message" && last.message.role === "assistant"
        ? text(last.message.content)
        : "";
    const outcome = report.match(
      /^Outcome:\s*(implemented|verified-unchanged|blocked|failed)\s*$/m,
    )?.[1];
    let problem = "";
    if (!outcome || !/^Evidence:\s*\S/m.test(report))
      problem = "Missing Outcome/Evidence handback.";
    else if (
      outcome === "implemented" &&
      (!baseline || baseline === (await snapshot(ctx)))
    )
      problem =
        "No changes or new commit since worker preflight; implementation is not established.";
    else if (
      outcome === "blocked" &&
      /serena|tool.*(?:unavailable|missing|access)/i.test(report) &&
      ![...failures].some((name) => report.includes(name))
    )
      problem =
        "Tool blocker has no named failed tool call. Try discovery/connection, then the documented fallback.";
    if (!problem) return;
    if (reminder)
      return {
        entries: [
          ...event.entries,
          {
            type: "custom_message",
            customType: "worker-contract-failed",
            content: `Outcome: failed\nEvidence: ${problem} Parent: inspect this task's activity before retrying; run completion is not task acceptance.`,
            display: true,
          },
        ],
        continue: false,
      };
    reminder = true;
    return {
      entries: [
        ...event.entries,
        {
          type: "custom_message",
          customType: "worker-handback-check",
          content: `${problem} Continue the task or report a precise, evidenced blocker. Do not restart or re-delegate it.`,
          display: false,
        },
      ],
      continue: true,
    };
  });
}
