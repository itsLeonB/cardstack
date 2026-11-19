# Concurrent frontend Vitest runs

## Decision

Measure unlocked runs with `maxWorkers: 2` in **both** apps first. Keep frontend's current cap; add the missing admin cap only if the measurements support it. If two workers fail under the supported load, test one worker in both apps. Remove the lock only after the chosen cap passes the checks below. This is a recommendation, not a measured fix: the lock and frontend cap have separate histories, and neither history proves that the present capped suites need a host lock. [R1][R2][H1][H2]

A fixed per-run cap permits concurrent worktrees; it does not guarantee success for an unlimited number of agents. Define the supported load as two and three simultaneous verification runs on the smallest supported machine, with room for the API, browser, builds and other work. This proposed limit follows from independent Vitest worker budgets and Docker's shared CPU scheduling, not from a new benchmark. [V1][V2][D1]

## Verified checkout and repo facts

Research used `/home/leon/Projects/itsLeonB/cardstack`, branch `docs/research-parallel-frontend-tests`, HEAD `fcc22c73e02f006e9d0f34e58271a2fec6f16bd5`; the initial worktree was clean. Serena's active project matched `git rev-parse --show-toplevel`. I read `AGENTS.md`, the Serena policy/manual and the research skill before source inspection. These are observations from this research session, not performance results. [O1]

| Area | Verified state | Source |
| --- | --- | --- |
| Lock | `serialized()` takes an exclusive `flock` on `${TMPDIR:-/tmp}/cardstack-verification.lock`; without `flock` it runs unlocked. Thus the default scope spans worktrees that use the same host and temporary directory, not every possible host execution. | [R1] |
| Entry point | Only `step_test` calls `serialized`; it wraps `bun run test` for frontend and admin. The script runs the selected packages in sequence and installs before each package's checks. `step_e2e` calls Playwright without this lock. | [R1] |
| Config | Frontend: jsdom, `testTimeout: 15_000`, `maxWorkers: 2`, one setup file. Admin: jsdom, `testTimeout: 15_000`, one setup file, no worker cap. Both package scripts are `vitest run`. Neither config sets a pool or concurrent test execution. | [R2] |
| Async waits | Both setup files set Testing Library `asyncUtilTimeout: 4000`. Their comments still mention Vitest's old 5 s timeout, although both configs now set 15 s. Frontend also stubs ResizeObserver; admin stubs matchMedia. | [R2][R3] |
| Versions and stack | Frozen lockfiles resolve frontend Vitest **4.1.11**, admin **4.1.12**, despite both manifests saying `^4`. Compose bind-mounts each checkout, shares the Bun package cache, and declares no CPU or memory limit for frontend-tools. The tooling image pins Bun 1.4.2 and Playwright 1.56.1. | [R4][R5] |

The setup helper in each historically affected file, `-collection-page.test.tsx:40–69` and `-collection-move.test.tsx:178–209`, creates a fresh QueryClient with query retries off, creates a memory router, renders RouterProvider and awaits `findByRole` for the loaded title. These are real asynchronous router/render waits, not proof that a timeout means CPU starvation. [R6][T1]

### Original changes and notes

| Change | Recorded evidence | What it does not prove |
| --- | --- | --- |
| `94c4f3ba` — lock | Adds the helper, test wrapper and testing guide note. The guide reports three queued runs taking 190 s versus 160 s concurrently. The commit attributes failures to CPU oversubscription. | No raw failure logs, worker counts, CPU trace or memory trace appear in these changed files; this is a historical report, not a current controlled comparison. [H1] |
| `3f1b3aef` — frontend cap | Reports collection page/move failures with an empty rendered body on a loaded 12-core host: one failure in four default runs, none in two capped runs; isolated, paired and serial runs passed. Reports 56 s default versus about 85 s capped and about 2.5× default when serial. | Passing two capped runs does not establish reliability, and passing alone does not rule out a timing-dependent bug. The commit's conclusion “contention, not shared state” is stronger than the evidence shown. [H2] |
| `89e9f34f` — RTL timeout | Reports first RouterProvider load at 1–2.5 s versus about 50 ms warm; under twelve busy-loop burners, seven failures in eight runs before the 4 s async timeout and zero in eight afterwards. | This supports sensitivity to load in that earlier test, not a diagnosis of every current failure. [H3] |
| `c7933b88` — Vitest timeout | Adds the frontend 15 s test timeout and a comment about cold route-module imports exceeding 5 s during parallel runs. | It does not change Testing Library's separate deadline. [H4][R3][T1] |

The existing test-audit report also distinguishes the Vitest lock from unlocked Playwright runs. I found these histories and notes through targeted Git searches; I did not establish that no other unpublished notes exist. [R7][O1]

### Cheap runtime observations, not a benchmark

An existing frontend-tools container answered `node -e` with Node **v22.20.0**, libuv **1.51.0**, `availableParallelism() = 12`, and `os.cpus().length = 12`. Its cgroup v2 files showed `cpu.max = max 100000`, `cpuset.cpus.effective = 0-11`, `memory.max = max`, and zero `nr_throttled`/`throttled_usec`. Docker inspect showed `NanoCpus=0`, `CpuQuota=0`, `CpuPeriod=0`, empty `CpusetCpus`, and `Memory=0`. These observations rule out an explicit leaf-container quota in that snapshot, not host contention or limits on other machines. [O2][D1][K1]

No test suite, CPU burner, build, package install or new container ran for this report. There are no new pass rates, wall times or memory measurements. [O1]

## Sourced platform facts

### Worker budgets are per Vitest run

Vitest has separate file-level and test-level concurrency. `maxWorkers` limits workers running files; tests within each file run sequentially unless concurrency is requested. `maxConcurrency` bounds concurrent tests inside a file and is not a replacement for `maxWorkers`. `fileParallelism: false` forces one worker; `maxWorkers: 1` can reduce file execution to one worker without disabling isolation. [V1][V2][V3]

Each independent Vitest invocation owns its own Pool and worker count. Three invocations capped at two therefore allow up to six test workers, plus runner processes, Vite transforms and runtime overhead; six workers are not a promise of six CPUs' total consumption. A percentage such as `50%` applies independently to each invocation, not to the remaining host capacity or the number of agents. [V2][V4]

**Version caveat:** the 4.1.11 maxWorkers documentation describes the run-mode default as all available parallelism, but its actual `resolveMaxWorkers()` uses `Math.max(availableParallelism - 1, 1)` when no cap is set; watch mode uses half. On the observed 12-CPU runtime, that source path permits eleven admin workers, subject to the number of test files. Use an explicit small integer rather than relying on wording in current online docs or default sizing. Confirm the resolved behavior of admin's 4.1.12 during measurement. [V2][V4][O2][R4]

Vitest 4 removed `minWorkers` and moved old `poolOptions` settings to top-level options; `maxThreads`, `maxForks`, `singleThread` and `singleFork` are not the options to introduce here. The documented default pool is `forks`, and the tagged resolver sets that default before its later fallback. Forks use child processes; threads use worker threads and have process-API/native-module limits. Keep the current pool while testing worker counts, so the experiment changes one cause at a time. [V5][V6]

### Container CPU counts are not a host-wide allocator

Node calls `availableParallelism()` an estimate and implements it through libuv. In the observed libuv 1.51.0 Linux source, the calculation checks CPU affinity and cgroup constraints; its Linux helper reads CPU quota/period, including cgroup v2 ancestor paths it can see. Thus “Node always sees host cores and ignores cgroups” is not sound for this runtime. Record Node and libuv versions, quota, affinity and the returned count together; do not infer them from `@types/node` or the Dockerfile alone. [N1][N2][O2]

Docker gives containers no CPU limit by default. `cpus`/quota sets a ceiling and can cause throttling; cpuset limits eligible CPUs; CPU shares are a relative weight under contention, not reserved capacity. Separate containers can each see the same eligible CPUs. A per-container quota is therefore not a cross-worktree admission policy, and adding tight quotas can make deadlines worse. Inspect parent/VM limits where relevant and the actual runtime rather than treating Compose's missing limit as unlimited physical capacity. [D1][N2][K1]

### Diagnose the deadline that actually failed

Testing Library's `findBy*` wraps `waitFor`; its upstream default timeout is 1 s, which this repo raises to 4 s. Async queries return promises; `waitFor` retries when the callback throws, not when it returns false. A 15 s Vitest test timeout cannot extend a 4 s Testing Library wait. Vitest hooks have their own timeout too. Record the exact failure type before changing any timeout. [T1][R3][V7]

The official user-event examples await user actions. For an actual failing test, inspect awaited actions, router readiness, mocked API completion, cleanup, timers, and unhandled errors before deciding that a slower CPU is the sole cause. An empty body shows the expected render did not appear by the deadline; it does not identify why. This is a diagnostic recommendation based on the affected helpers and the async contracts, not a finding that those helpers contain a proven bug. [T1][T2][R6][H2]

Vitest's tagged profiling guide separates transform, setup, import, environment and test time. Its import timing excludes dynamic imports made inside tests: those costs can consume the test's own deadline. Setup files run before each test file; isolation also repeats environment/module work. Use the installed version's duration summary and import profiling when needed, not wall time alone. Current online performance docs describe features and timings that need not match these pinned patch versions; no proposed speedup here depends on `vitest doctor` or new cache defaults. [V8][V9][V10]

## Hypotheses and unmeasured claims

| Hypothesis | Evidence to collect before calling it a cause |
| --- | --- |
| Too many workers across worktrees | Compare identical unlocked runs at workers 1/2/4 with synchronized starts; collect CPU use, wall time, failure deadlines and runnable pressure. Historical loaded-host failures make this plausible, not proved. [H2][V4] |
| Cold environment/import/setup cost | Compare first-run and repeat phase timings; inspect slow imports only after identifying the dominant phase. Distinguish a new process's cold import from a genuinely empty disk cache. [V8][V9] |
| Async render/test race | Retain the first failing assertion, DOM, warnings and exact test; run it alone and beside its usual neighbors under the same load. Check awaited readiness rather than adding sleeps. [R6][T1][T2] |
| Memory pressure or CPU throttling | Compare memory usage/peak and memory.events, cpu.stat deltas, and host pressure with the failure time; a timeout alone proves neither. The observed unlimited container is not evidence about past runs. [D1][K1][O2] |

## Ranked smallest changes

1. **First candidate: keep frontend at two; cap admin at two; measure unlocked.** Likely code change: add `maxWorkers: 2` under `test` in `admin/vite.config.ts`. After acceptance, replace `RUN_DIR=$pkg serialized run bun run test` with `RUN_DIR=$pkg run bun run test` in `scripts/verification/frontend.sh`, delete the now-unused `serialized()` helper from `scripts/verification/lib.sh`, and update the lock paragraph in `docs/agents/testing.md`. A Git search found no other script caller. Leave package scripts, test deadlines, isolation, pool and E2E unchanged. This is the smallest candidate addressing the uncapped independent run. [R1][R2][V2][V4][O1]
2. **If two workers still fail: use one in both apps, with isolation intact, and repeat.** Change `maxWorkers` only; do not combine it with `isolate: false`. This exchanges per-run throughput for a smaller combined budget without blocking another worktree. It still needs measurements on the smallest supported host, and it still cannot promise unlimited concurrent runs. [V2][V3][V4]
3. **If failures remain at one: repair the measured cause before removing the lock.** Fix a demonstrated readiness/cleanup issue in the failing tests, or reduce a measured import/environment cost. Keep the lock until that change passes the same matrix. A pool comparison (`--pool=threads` versus `--pool=forks`) is a later controlled experiment, not an initial migration. [T1][T2][V6][V8]

For focused agent checks, use a filename filter with a small explicit cap, for example `docker compose exec -T -w /workspace/frontend frontend-tools bun run test --maxWorkers=1 -collection-page.test.tsx`. Vitest also supports `related --run` and `--changed`, but related-file selection depends on imports it can resolve and does not replace the full gate. The script's current `test` step accepts no forwarded Vitest arguments, so use Compose directly for focused work and keep the documented verification script as the final gate. [V11][R1][R8][B1]

For measured environment cost, possible narrow candidates are `frontend/src/lib/catalog-search.test.ts` and `frontend/e2e/support/clerk-credentials.test.ts`: the inspected code has no direct DOM references. Vitest supports `// @vitest-environment node` per file. Verify their dependency graphs and setup behavior before adopting it; “all .test.ts files are pure” is not established. Both global setup files import React Testing Library, and admin's setup uses `window`, so a blanket node-environment change is not safe as written. This is an optional follow-up, not required for lock removal. [R3][R9][V12]

### Not the default fix

| Option | Reason to defer |
| --- | --- |
| Blanket longer timeouts or retries | They do not reduce worker demand or identify a missed async transition; the repo already has distinct 15 s/4 s deadlines. Retain first-attempt failures and do not let retries turn the acceptance matrix green. [R2][R3][T1][V7] |
| Disable isolation | Files can then share module/global state; repeated setup and browser mocks need a separate audit. A worker cap does not require this trade-off. [V3][V9][R3] |
| Migrate to happy-dom or VM pools | Vitest documents missing APIs in happy-dom and VM/ESM memory and realm limits. These change test semantics and require more checking than a worker cap. [V6][V12] |
| Add a semaphore scheduler or quotas | A semaphore needs cross-process ownership, cancellation and admission rules; that is additional scope, not needed to try the existing worker option. Quotas throttle rather than reserve CPU. Revisit admission control only if measured supported loads cannot work within a fixed per-run budget. [V4][D1] |

## Reproducible measurement plan — not executed

### Matrix

Use existing, distinct worktrees at the same commit, with their own Compose projects and frozen dependencies installed before timing. Keep source unchanged, do not install/build during the isolated Vitest measurements, keep the default pool/isolation/timeouts, and avoid overlapping two jobs inside the same worktree. Record all other active workloads. This protocol isolates worker-count effects from the script's install and full-gate steps. [R1][R5][V8]

| Simultaneous worktrees | Current settings baseline | Explicit workers per instance | Cache states | Status |
| --- | --- | --- | --- | --- |
| 1 | frontend 2; admin default | 1, 2, 4 | fresh process first run; immediate repeat | Unmeasured |
| 2 | frontend 2; admin default | 1, 2, 4 | fresh process first run; immediate repeat | Unmeasured |
| 3 | frontend 2; admin default | 1, 2, 4 | fresh process first run; immediate repeat | Unmeasured |

Run each cell for frontend alone, admin alone, and the production package order frontend → admin in each worktree. Also stagger one mixed batch so an admin instance overlaps a frontend instance: synchronized package order can hide that combination. Start with one screening batch per cell; stop a clearly failing/high-pressure cell and preserve its logs. Repeat viable workers 1/2 candidates ten times per load and cache state. These repetition counts are proposed acceptance policy, not a statistical guarantee. [R1][V4]

“Cold” must state what is cold. Every `vitest run` starts a fresh process; an immediate repeat can still reuse OS/filesystem caches. Label a new-worktree/no-generated-test-cache run separately from a warm-cache run. Do not drop host-wide caches or delete dependency/cache folders while other agents work. Record retained cache paths and whether dependency installation happened before the first sample. This is experimental control, not a claim that a particular cache caused the bug. [V8][R5]

### Commands for one explicit-worker cell

Set `WT1`, `WT2`, `WT3` to three already prepared worktrees; choose `N=1`, `2` or `3`, `W=1`, `2` or `4`, and `PKG=frontend` or `admin`. Run from Bash. The command deliberately bypasses `step_test`'s lock **only for this experiment**; Bun forwards arguments after the script name. For the current-settings baseline, omit `--maxWorkers="$W"`. Repeat the cell after the first batch finishes. [R1][B1][V2]

```bash
N=3 W=2 PKG=frontend
roots=("$WT1" "$WT2" "$WT3")
out="$(mktemp -d /tmp/cardstack-vitest-measure.XXXXXX)"
pids=()
for ((i=0; i<N; i++)); do
  (
    set +e
    start=$(date +%s)
    docker compose --project-directory "${roots[$i]}" exec -T \
      -w "/workspace/$PKG" frontend-tools \
      bun run test --maxWorkers="$W" >"$out/$i.log" 2>&1
    rc=$?
    printf '%s,%s,%s,%s,%s\n' "$i" "$PKG" "$W" "$rc" "$(( $(date +%s)-start ))" \
      >"$out/$i.result.csv"
    exit "$rc"
  ) &
  pids+=("$!")
done
failed=0
for pid in "${pids[@]}"; do wait "$pid" || failed=1; done
printf 'logs=%s failed=%s\n' "$out" "$failed"
```

The subshell disables exit-on-error so a failed Vitest run still writes its result. The seconds field is host-side command wall time including Docker exec, not Vitest's summed phase timings. Preserve every run's full log. This is measurement protocol, not a new repo script.

For an apples-to-apples locked baseline, replace the Docker command in the same loop with `bash -c 'source "$1/scripts/verification/lib.sh"; RUN_DIR="$2" serialized run_in frontend-tools bun run test' bash "${roots[$i]}" "$PKG"`, keeping the same log redirection and result handling. This uses current package settings and the actual lock, but excludes installation just like the unlocked experiment. Measure complete script runs separately, because they include installation. [R1]

Before/after each batch, collect container IDs with `docker compose --project-directory "$WT1" ps -q frontend-tools` for each selected worktree. Sample `docker stats --format '{{.Name}},{{.CPUPerc}},{{.MemUsage}},{{.PIDs}}' <ids>` from another terminal through the batch. Read `/sys/fs/cgroup/cpu.stat`, `cpu.max`, `memory.current`, `memory.peak` if present, and `memory.events` inside each container before/after; use counter deltas, because a long-lived container's peak and counters include earlier work. Capture host CPU/memory pressure too. Docker CLI memory subtracts cache, so retain raw cgroup memory alongside its display. [D2][K1]

Record this row for **every** invocation, plus batch makespan: `commit, package, runtime_versions, N, W_or_default, pool, isolation, cache_state, repetition, exit_code, passed_tests, failed_tests, failed_test_and_deadline, wall_s, transform_ms, setup_ms, import_ms, environment_ms, tests_ms, cpu_usage_delta_usec, nr_throttled_delta, throttled_usec_delta, sampled_peak_memory_bytes, memory_events_delta, other_load`. Use “not available” rather than inventing absent metrics. Container CPU deltas include other commands in that container; collect them only when the tooling container is otherwise idle. [V8][D2][K1]

### Acceptance before lock removal

1. Both apps must pass without retries in repeated one-, two- and three-worktree cold/warm batches, including mixed frontend/admin overlap and the two historically affected files. Proposed minimum: ten batches per viable cell; retain every failure rather than discarding it as a flake. Historical two-pass evidence is not enough for this policy. [H2]
2. Compare total batch makespan and worst individual wall time with the controlled locked baseline above at the same load; also compare complete locked/unlocked `scripts/verification/frontend.sh test` runs separately. Choose the smallest worker count that meets the agreed elapsed-time budget, not simply the fastest isolated run. No new elapsed-time budget was supplied for this research. [R1][H1]
3. Require no new OOM events or sustained memory/CPU pressure correlated with missed deadlines on the smallest supported host. If one worker still fails, investigate its async/import path before accepting a timeout change. Cgroup throttling alone is not proof of a test bug, and zero leaf throttling is not proof of spare host CPU. [D1][K1][T1][V8]
4. After the candidate passes isolated Vitest checks, run the unchanged full verification gate concurrently at the supported load. It includes installs, builds and unlocked Playwright work that the unit-only matrix excludes; diagnose E2E failures separately. Update the documented concurrency limit and retain the measured data before deleting the lock. [R1][R7][R8]

## Sources and evidence boundaries

**[O1] Session observations:** `pwd`, `git status --short --branch`, `git branch --show-current`, `git rev-parse HEAD`, `git rev-parse --show-toplevel`; Serena `initial_instructions`; targeted Git history/pattern searches and read-only source inspection. No performance test ran. These observations refer only to this research checkout and session.

**[O2] Runtime probe:** `docker compose exec -T frontend-tools node -e` printed `process.version`, `process.versions.uv`, `os.availableParallelism()`, `os.cpus().length` and the four named cgroup files; `docker inspect` printed the five named HostConfig resource fields. Snapshot values appear above; no load measurement accompanied them.

**[R1] Local scripts:** `scripts/verification/lib.sh`, especially `serialized()` and `run_in()`; `scripts/verification/frontend.sh`, especially package loop, `install`, `step_test`, `step_e2e`; `git grep -n serialized -- scripts` at HEAD above.

**[R2] Local configs/scripts:** `frontend/vite.config.ts`, `admin/vite.config.ts`, `frontend/package.json`, `admin/package.json` at HEAD above.

**[R3] Local setup:** `frontend/src/test-setup.ts:1–14`, `admin/src/test-setup.ts:1–18` at HEAD above.

**[R4] Locked versions:** `frontend/bun.lock`, `vitest@4.1.11` entry; `admin/bun.lock`, `vitest@4.1.12` entry; both manifests and the script's `bun install --frozen-lockfile`.

**[R5] Local stack:** `compose.yaml`, `x-tools`, `frontend-tools` and named caches; `frontend/tooling.Dockerfile`; `docs/agents/testing.md`, “The stack”.

**[R6] Affected test helpers:** `frontend/src/routes/_authenticated/collections/$collectionId/-collection-page.test.tsx`, `renderCollectionPage`; neighboring `-collection-move.test.tsx`, `renderCollectionPage`; retrieved through Serena symbol reads.

**[R7] Prior audit:** `.scratch/test-audit/report.md`, section “6a. Parallelism: what the scripts actually constrain”. Its historic E2E findings are not new Vitest measurements.

**[R8] Gate policy:** `docs/agents/testing.md`, opening paragraph and “Run the checks”.

**[R9] Narrow environment candidates:** Serena pattern inspection of `frontend/src/lib/catalog-search.test.ts`, `frontend/src/lib/catalog-search.ts`, `frontend/e2e/support/clerk-credentials.test.ts`; no node-environment run performed.

**[H1] Lock history:** `git show 94c4f3ba0a46c4fd2764fa84481d2eac68850116`; associated measured-cost paragraph in `docs/agents/testing.md`.

**[H2] Worker history:** `git show 3f1b3aef442163a2883ba67e571f4f8d2f6e2ede`; commit message and frontend config diff. The causal conclusion belongs to that commit, not this report.

**[H3] Async history:** `git show 89e9f34fd48274c27694da8561c329a16b4b9b4f`; commit message and setup/config diff.

**[H4] Test-timeout history:** `git show c7933b88184dcaba0e4648a7a98e2210bc71a943`; frontend config diff.

**[V1] Vitest 4.1.11 parallelism guide:** https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/guide/parallelism.md

**[V2] Vitest 4.1.11 maxWorkers docs:** https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/config/maxworkers.md

**[V3] Vitest 4.1.11 config resolver:** `fileParallelism`, `maxWorkers` and pool defaults, https://github.com/vitest-dev/vitest/blob/v4.1.11/packages/vitest/src/node/config/resolveConfig.ts ; corresponding admin tag: https://github.com/vitest-dev/vitest/blob/v4.1.12/packages/vitest/src/node/config/resolveConfig.ts

**[V4] Vitest 4.1.11 worker implementation:** `resolveMaxWorkers`, `createPool` and `Pool`, https://github.com/vitest-dev/vitest/blob/v4.1.11/packages/vitest/src/node/pool.ts and https://github.com/vitest-dev/vitest/blob/v4.1.11/packages/vitest/src/node/pools/pool.ts

**[V5] Vitest 4 migration:** “Pool Rework” and removed options, https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/guide/migration.md

**[V6] Vitest 4.1.11 pools:** https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/config/pool.md

**[V7] Vitest 4.1.11 deadlines:** https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/config/testtimeout.md and https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/config/hooktimeout.md

**[V8] Vitest 4.1.11 profiling:** timing definitions and import profiling, https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/guide/profiling-test-performance.md

**[V9] Vitest 4.1.11 setup:** https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/config/setupfiles.md

**[V10] Performance docs, version comparison:** tagged https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/guide/improving-performance.md versus live https://vitest.dev/guide/improving-performance ; do not assume the live page describes the installed version.

**[V11] Vitest 4.1.11 CLI:** filename filters, `related`, `--changed`, https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/guide/cli.md

**[V12] Vitest environments:** per-file overrides, https://github.com/vitest-dev/vitest/blob/v4.1.11/docs/config/environment.md ; jsdom/happy-dom API trade-offs, https://vitest.dev/guide/environment.html

**[N1] Node 22.20.0 os API:** `os.availableParallelism()`, https://nodejs.org/download/release/v22.20.0/docs/api/os.html#osavailableparallelism

**[N2] libuv 1.51.0 Linux implementation:** `uv_available_parallelism`, https://github.com/libuv/libuv/blob/v1.51.0/src/unix/core.c ; `uv__get_constrained_cpu`, https://github.com/libuv/libuv/blob/v1.51.0/src/unix/linux.c

**[D1] Docker resource limits:** defaults, CPU quota/cpuset/shares and memory risks, https://docs.docker.com/engine/containers/resource_constraints/

**[D2] Docker stats:** CPU/memory/process reporting and Linux cache subtraction, https://docs.docker.com/reference/cli/docker/container/stats/

**[K1] Linux cgroup v2:** cpu.stat, cpu.max, memory.current/peak/events and hierarchy, https://docs.kernel.org/admin-guide/cgroup-v2.html

**[T1] Testing Library async API:** `findBy`, `waitFor`, promise/error retry semantics and default deadlines, https://testing-library.com/docs/dom-testing-library/api-async/

**[T2] Testing Library user-event:** setup and awaited interaction examples, https://testing-library.com/docs/user-event/intro/

**[B1] Bun run:** script arguments and Node shebang behavior, https://bun.sh/docs/cli/run ; Bun invoking `vitest run` does not by itself mean the tests use Bun's test runner or Bun runtime.
