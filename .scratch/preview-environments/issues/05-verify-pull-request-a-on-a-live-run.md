# 05: Verify the preview infrastructure on a live PR run

**What to build:** Proof that the first pull request works end to end on a real pull request, since the workflow cannot be tested locally and workflow changes in a PR take effect for that PR. This needs the repository's Neon, Railway and Vercel credentials, so it is done by the maintainer.

**Blocked by:** 02, 03, 04.

**Status:** done

- [x] Two consecutive pushes to the PR each produce exactly one API deployment.
- [x] Data created in the preview after the first push is still there after the second push (the Neon branch is kept).
- [x] The PR environment has none of the remove-strategy variables and has `APP_ENV=preview`.
- [x] The frontend alias resolves, and a development Clerk sign-in succeeds against the API (the allowed origin matches).
- [x] The Railway domain existed before the API's first deployment; if it did not, record how the workflow had to change.
- [x] A PR environment whose provisioning run is skipped or cancelled refuses to boot with the guard's message instead of running against production.
- [x] Closing the PR deletes the Neon branch, the alias, the deployments and the branch-scoped frontend variables.
- [x] Any workflow fix needed is committed on the same pull request, and the outcome is recorded in this ticket's comments.

## Comments

### The verdict

The run happened and passed: pull request #69 (`feat/preview-sign-in-adoption`, merged 2026-10-31) was provisioned by the workflow as it stands in `main` after `5ecc6f5`, and its preview API deployed and served. Every box above is ticked on the maintainer's report of that run. Below is the evidence retained in the repository, and which of it is a lasting artifact as opposed to a claim about a run nobody can re-inspect now.

### What the repository still shows

- The workflow's current version is what ran: PR #69's comment carries the current wording and the `cardstack-pr-69.vercel.app` alias, and the three preview-workflow commits of 2026-10-31 (`77b72a9`, `9e63527`, `462beb2`, `5ecc6f5`) all landed before the merge. Every `Preview Environments` run in `gh run list` is green.
- The alias is assigned and later freed: `https://cardstack-pr-69.vercel.app` answers 404 today, after the merge.
- The Railway domain exists before the API deploys, and independently of it: PR #72's Railway comment lists `https://api-cardstack-pr-72.up.railway.app` next to an api row whose status is `Skipped`. The API never deployed there, and the domain was still allocated.
- The comment is updated in place: PRs #68, #69 and #72 each hold exactly one comment starting with `### Preview environment`, across several pushes each.
- The boot guard is covered by unit tests from ticket 04, not only by the live run.

### What no longer has an artifact

The Neon branch contents across two pushes, the variable set after provisioning, and the per-push deployment count all died with each PR's Railway environment and Neon branch. Re-opening the question costs a new pull request: `gh run list --workflow=preview-environments.yml` plus the Railway environment's deployment list is enough. The "exactly one deployment per push" claim is re-checked as a box in ticket 07, because that ticket changes what triggers a deployment.
