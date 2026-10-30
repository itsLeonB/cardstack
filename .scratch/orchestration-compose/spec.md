# Spec: Orchestration review flow and a compose-based tooling stack

**Status:** ready-for-agent

Sources: the retrospective of the ticket 06 session (PR 65), the grilling session of 2026-10-30, and the research in `.scratch/agent-tooling/research/harness-child-branch-naming.md`. It follows the earlier orchestration spec in `.scratch/agent-tooling/spec.md`, which is done.

## Problem Statement

The maintainer runs development through an orchestrator that delegates to backend and frontend workers in git worktrees. The ticket 06 session showed that the written workflow does not hold up in practice, in four ways.

First, the review rules are unclear. The orchestrator skipped the required code review and used the advisor as the only independent pass, and the docs make the two roles overlap: code review is a per-child loop in some places and an advisor consult in others. Nobody can say from the docs when a review runs, who runs it, or how many times.

Second, the docs fight the harnesses. The agent-facing Serena doc hardcodes one harness's tool names and call form, which was wrong in the session (the real names carried a different prefix, and the server had to be connected first). The merge helper's documented flag syntax does not match its parser, so a two-directory scope failed. The docs require worker branches named by convention, but some harnesses create and name the worker's branch themselves and offer no way to choose it.

Third, there is no dependable test environment. Backend verification needs a Postgres with the vector extension, which the local one lacked, so the first backend worker ended blocked. The lint tool could not run in the sandbox. Each CI workflow rebuilds its own stack by hand, so what CI runs and what an agent runs can drift apart. Nothing runs the browser suite against anything close to the production build.

Fourth, the maintainer found a user-facing bug on the preview that no automated check could have caught, because the only coverage was mocked-network screen tests.

## Solution

Agents and CI use one stack and one set of entrypoints. A single compose file brings up the database, the API, the user app and the Admin app, and every lint, test, build and browser run executes inside tooling containers with `compose exec`. The applications run from production-style images so the development environment stays close to what ships. The verification scripts become the one entrypoint for agents and for CI, with steps that CI can run as separate, identically named jobs.

The review flow is defined once. Code review runs once per implementation loop, by whoever did the implementing, and once more by the orchestrator after the merge. The advisor is a separate role, consulted only by the orchestrator when it must choose between options, never as a routine check.

The docs stop depending on any harness. The agent Serena doc says how to find and connect Serena's tools by capability, the human Serena doc covers the two clients the team uses, and the branch rule becomes "the name is a request, the path is the contract". The merge helper learns to take a component name and a worktree path so the orchestrator does not copy branch names by hand.

## User Stories

1. As the maintainer, I want the review flow written once, so that every session runs the same reviews in the same order.
2. As the maintainer, I want code review to run after every implementation loop, so that no change reaches the integration step unreviewed.
3. As the maintainer, I want a worker to review its own diff once before it reports, so that the orchestrator receives work that already passed one review.
4. As the maintainer, I want the worker's report to say which review findings were fixed and which were discarded, so that I can see the review happened.
5. As the maintainer, I want the orchestrator to run code review once on the merged feature branch, so that cross-component problems are caught after integration.
6. As the maintainer, I want fixes made from a review finding not to be reviewed again, so that review loops end.
7. As the maintainer, I want a new request from me to start a new loop with its own review, so that follow-up changes are not skipped.
8. As an orchestrator, I want to review small tasks myself on my own model, so that small changes do not need delegation.
9. As an orchestrator, I want the advisor only for choices between options, so that I do not spend a stronger model on routine checks.
10. As an orchestrator, I want to consult the advisor when a worker's question is not answered by the ticket or the ADRs, so that I do not guess.
11. As an orchestrator, I want to consult the advisor when the same error survives two different fixes, so that I stop iterating blindly.
12. As an orchestrator, I want the advisor message to carry the options, my leaning and what to read, so that the advisor, which sees only that message, can answer without the conversation.
13. As an orchestrator, I want the advisor to open with a recommended option and its reasons, so that I can act on it quickly.
14. As an orchestrator, I want to surface a conflict when my own evidence contradicts an advisor claim, so that I follow the advice without following it blindly.
15. As a worker, I want to ask the orchestrator instead of the advisor when I am unsure, so that the roles stay separate.
16. As a worker, I want the task message to tell me to run code review once before reporting, so that I know the exact step.
17. As an agent in any harness, I want the Serena doc to tell me to find the tools by Serena's own names, so that a different prefix does not stop me.
18. As an agent whose Serena tools are not listed, I want to be told to connect or enable the server first, so that I do not fall back to built-in file tools by accident.
19. As an agent, I want to call the project-activation tool only if it exists, so that I do not fail on a setup where the project is already active.
20. As a maintainer using Claude Code or pi, I want a human doc that explains how Serena is wired in each, so that I can fix a broken setup without reading agent instructions.
21. As a maintainer, I want the pi tool naming and the connect step documented in the human doc, so that the surprise from the last session is recorded where a human looks.
22. As an orchestrator, I want to pass the merge helper a component name, so that the directory list lives in one place.
23. As an orchestrator, I want the merge helper to accept several scope values after one flag, so that the documented syntax works.
24. As an orchestrator, I want to give the merge helper a worktree path, so that it reads the live branch itself.
25. As an orchestrator, I want the merge helper to refuse a detached HEAD, so that a worker's stray checkout cannot be merged by accident.
26. As an orchestrator, I want to take the worker's worktree path from the harness and read the real branch from git, so that I do not depend on a name I could not choose.
27. As an orchestrator, I want the branch naming convention to apply only where I can choose the name, so that the docs never ask for something a harness forbids.
28. As an agent, I want to run `compose up` once and then run all tooling with `compose exec`, so that I never install toolchains or provision a database myself.
29. As an agent, I want a Postgres with the vector extension in the stack, so that backend tests never block on a missing extension.
30. As an agent, I want lint, build, typecheck and unit tests to run in tooling containers, so that results do not depend on the host.
31. As an agent, I want each worktree to get its own isolated stack, so that parallel workers do not collide on ports or share a database.
32. As an agent, I want the stack to publish no fixed host ports by default, so that two worktrees can run at once.
33. As a maintainer, I want an opt-in override that publishes ports and rebuilds with localhost addresses, so that I can open the app in my own browser.
34. As an agent, I want the verification scripts to bring the stack up themselves, so that one command runs a whole component check.
35. As an agent, I want the verification scripts to accept a step name and to run all steps with no argument, so that I can run one step while debugging.
36. As a CI maintainer, I want each CI job to call the same verification entrypoint, so that agents and CI run identical commands.
37. As a CI maintainer, I want the existing job names to stay unchanged, so that branch protection keeps working.
38. As a CI maintainer, I want steps that need no database to start only the tooling container, so that lint and typecheck jobs stay fast.
39. As a CI maintainer, I want image layers and package caches restored between runs, so that a warm run is not slower than today's.
40. As the maintainer, I want the API and both apps to run from production-style images, so that the development environment is as close to production as possible.
41. As the maintainer, I want the existing production backend image left unchanged, so that nothing about what ships is affected.
42. As an agent, I want the migration to run from the same image as the API, so that the stack exercises the real migration binary.
43. As an agent, I want a seed profile that loads the end-to-end fixture, so that the browser suite has deterministic data.
44. As an agent, I want the browser suite to run in the frontend tooling container against the built app, so that it tests the production build.
45. As an agent, I want the end-to-end step to rebuild changed images automatically, so that I never test stale code.
46. As an agent, I want unit tests, lint and typecheck to need no rebuild, so that the fast loop stays fast.
47. As the maintainer, I want the user app and the Admin app each served by a small script that reads its own `vercel.json`, so that headers and rewrites cannot drift from Vercel's.
48. As an end user of the deployed app, I want nothing to change, so that this work carries no production risk.
49. As the maintainer, I want the signed-in browser specs to run in the stack when Clerk credentials exist and skip with a reason when they do not, so that fork pull requests still pass.
50. As the maintainer, I want the stack to name its own in-network origins for the API's allowed client addresses, so that sign-in works inside the stack.
51. As a maintainer, I want the docs to describe the new stack and drop the instructions for provisioning Postgres by hand, so that there is one way to get an environment.
52. As a reviewer, I want compose and workflow YAML to pass the repo's YAML linter, so that configuration errors are caught mechanically.
53. As the maintainer, I want the docs never to name a specific orchestration product, so that they stay valid when the harness changes.
54. As an orchestrator, I want the build-time-only address in the app images documented, so that I know why a changed API address needs a rebuild.

## Implementation Decisions

The work is two deliverables, each its own branch from the main branch and its own pull request, independent of the open pull request for ticket 06.

**Deliverable 1: review flow and docs, owned entirely by the orchestrator** (docs, the tools directory, agent role files and the repo-level agent instructions all sit outside the component directories).

- Code review runs once per implementation loop. A loop is one implementation pass started by a task or a user input. The sequence is implement, review once, fix the findings, done.
- A small task has one loop, run by the orchestrator on its own model. A large task has three loops: one per worker, run by that worker on the worker's model before it reports, and one by the orchestrator on the merged feature branch.
- Findings from the integration review are fixed by a fresh worker or, outside the component directories, by the orchestrator. Those fixes are not reviewed again. Verification reruns after them. The earlier "stop after two rounds" limit goes away.
- The worker report gains a line stating which review findings were fixed and which were discarded. The task message gains a review item telling the worker to run the review once before reporting. The existing statement that a child never reviews its own work is removed.
- The advisor is called only by the orchestrator, and only at decision points: choosing between approaches, a worker question the ticket and ADRs do not answer, and a failure that survives two different fixes. It is not called to confirm finished work. Short tasks do not use it.
- The advisor message is self-contained, because the advisor sees only that message: the options, the orchestrator's leaning, and what to read. The reply opens with the recommended option and the reasons. If the orchestrator's own evidence contradicts a specific claim, it surfaces the conflict, sends one reconcile message, then asks the maintainer. The advisor role file changes from judging the work to judging a decision. The advisor principles follow the Claude Code advisor documentation: timing is the caller's choice, the output is guidance and not an order, and it adds little on short tasks.
- Workers ask the orchestrator when unsure, never the advisor. The repo-level agent instructions and both component role files change to match.
- The agent-facing Serena doc becomes harness-neutral. It says to find Serena's tools by Serena's own names whatever prefix a harness adds, to connect or enable the server through the harness's own MCP mechanism when no Serena tool is listed and only then fall back to built-in tools, and to call the project-activation tool only if it exists. The tool mapping table stays, keyed by Serena's own names.
- The human-facing Serena doc covers Claude Code and pi only, including pi's tool naming and the connect step. No other harness gets a section.
- The branch rule is rewritten: the branch name is a request and the worktree path is the contract. The orchestrator records the worker's worktree path from the harness, reads the live branch from git, and passes that to the merge helper. The `<type>/<name>` convention applies only where the caller can choose the name. Workers never rename their branch. The docs do not name any orchestration product.
- The commit naming convention stays, set by the task message. Fix-up workers have no branch naming requirement.
- The merge helper gains three things. The scope flag accepts several values after one flag, and a repeated flag keeps working. A component flag expands a component name to its directories, with the frontend component covering both frontend packages, so the map lives only in the helper. A worktree flag takes a worktree path, resolves the live branch and refuses a detached HEAD. The branch argument stays as an alternative. The cleanup subcommand gains the same worktree form.

**Deliverable 2: the compose stack, a large task by the orchestration docs** (new files sit in component directories, so workers write them and the orchestrator owns the rest).

- One compose file at the repository root defines the stack: a database using the pgvector image the tests already require, a one-shot migration from the production backend image, the API, the user app and the Admin app from production-style images, and one tooling container per component. A seed profile loads the end-to-end fixture. Playwright is part of the frontend tooling container, not its own service.
- The production backend image is unchanged and is what the stack runs. The migration uses the same image's migrate binary.
- New production-style images for the user app and the Admin app each have a build stage that runs the app's production build with the API address and the Clerk publishable key as build arguments, and a runtime stage that serves the built client output. The runtime is a small Bun script, one per app, that reads that app's `vercel.json` and applies its rewrites and headers with URL patterns, so the rules have a single source of truth. It serves a real static file when one exists and the shell page otherwise.
- Each component has a separate tooling image kept with the component. The backend tooling image is a glibc Go image with a C compiler, because the race detector needs cgo, and the pinned lint tool. The frontend tooling image is the official Playwright image at the version `package.json` pins, plus bun, and serves both the user app and the Admin app. The tooling images bind-mount the repository, run as the host user so no root-owned files appear, and keep a long-lived process so `compose exec` works.
- The default stack is isolated per worktree. The compose project name derives from the worktree directory, no host ports are published, and containers reach each other by service name. The app images bake in-network addresses at build time, and the API's allowed client addresses list the in-network app origins. A human who wants a browser adds a gitignored override that publishes ports and rebuilds the images with localhost addresses.
- Because the apps run from built images there is no hot reload. Source edits reach a running app only through a rebuild. The end-to-end step rebuilds automatically. Lint, typecheck and unit tests run against the bind-mounted source and never need a rebuild.
- The verification scripts remain the single documented entrypoint for each component. Each accepts a step name and runs all steps in order when given none. They start the stack themselves, starting only the tooling container for steps that need no database.
- The four CI workflows for the backend, the user app, the Admin app and the browser suite move onto the same scripts. Each keeps its parallel jobs and exact job names so required checks keep working. CI builds the images with layer caching and restores the Go and bun package caches from bind-mounted cache directories. The preview environment workflow and the convention-check workflow are untouched.
- The signed-in browser specs read the Clerk variables from the existing env files when present and skip with a stated reason when absent, as today. The maintainer believes the Clerk development instance accepts any origin. That is untested and is verified during this work, with a fallback to guest-only specs in the stack if the origin is refused.
- Worker delegation: a backend worker writes the backend tooling image only. A frontend worker writes the two app images, both serve scripts, and the frontend tooling image. The two workers run in parallel and each reviews its own diff once. After both merge, the orchestrator writes the compose file, the verification and bootstrap scripts, the workflow changes and the docs, runs the whole stack locally, then runs the integration review.
- The bootstrap scripts shrink to copying the gitignored env files and starting the stack. The host keeps its language toolchains for Serena's language servers, and the Serena setup is unchanged.
- The testing and environment docs are rewritten compose-first. The manual Postgres provisioning instructions and the uuid shim are removed or reduced to a short fallback pointer.

## Testing Decisions

A good test checks external behaviour through the highest available seam and does not assert implementation details. The ideal is few seams.

- **Merge helper.** The existing entry-point seam in its test file, which builds real git repositories in a temporary directory, is extended. New cases: several scope values after one flag, a repeated scope flag, the component flag expanding to the right directories, the worktree flag resolving the live branch (including after a rename), a detached HEAD being refused, and cleanup by worktree path. Prior art: the existing merge and cleanup cases in that file.
- **Serve scripts.** One new seam, a request-in, response-out test of the serve handler, run against the real `vercel.json` of each app. It covers the fallback to the shell page, the no-index headers on exactly the paths the file names, a real static file winning over the fallback, and the Admin app's no-index header on every path. It runs under each package's existing unit test command. Prior art: the existing unit tests beside the frontend tools.
- **Compose stack.** No new unit tests. The seam is the verification entrypoints running through compose: the stack passes when the existing backend tests, the frontend and Admin unit tests, lint, typecheck, build and the full Playwright suite pass unchanged, locally and in all four CI workflows, with job names unchanged. The orchestrator runs the whole stack locally before merging, including a cold start, a step run on its own, and a rebuild after a source edit.
- **Docs and agent files.** No automated seam. They are checked by the integration code review, the existing convention check that bounds bullet length, and the repo's YAML linter for the compose file and workflows, per the repo rules. Markdown stays one line per paragraph, list item and table row.

## Out of Scope

- Building the production backend image in CI. Nothing builds it today, so a broken production Dockerfile would show only at deploy time. This is a worthwhile follow-up and is not part of this work.
- Aligning the production backend image's Go version with the module file. It is left alone because that image is unchanged.
- The flaky race-detector test run in the backend. Compose does not fix it.
- Moving the preview environment workflow, which provisions hosted services for humans, onto compose.
- Hot reload and any development-server mode in the stack. Humans who want it keep running the dev servers natively.
- Documentation or wiring for harnesses other than Claude Code and pi.
- New product features, new browser specs for the collection selection flow, and a test that every catalog-search route is rate-limited. These came out of the retrospective and are separate tickets.
- Changing the open pull request for ticket 06.

## Further Notes

- The research behind the branch rule is in `.scratch/agent-tooling/research/harness-child-branch-naming.md`. Its findings are verified in source for the one harness that mints branch names itself, and not tested live. It recommends the same rule adopted here.
- Risks to watch: the untested Clerk origin; a slower first CI run on cold caches; the required check names; the browser suite moving from the dev server to the built app, which may surface differences; and Haiku workers writing infrastructure files, which needs a careful review.
- The two Dockerfile pairs for the apps are an imitation of the hosted platform, not what ships. The serve script reading `vercel.json` is what keeps them honest.
- Order of work: deliverable 1 can start immediately. Deliverable 2 starts with the two parallel workers, then the orchestrator's part.
- Rough effort: deliverable 1 about two hours; deliverable 2 a day or two across worker runs, reviews and CI iterations.
