# 01: Reconcile script and preview variable manifest

**What to build:** A checked-in manifest classifying every variable the API can have in a preview (inherit, override, derive or remove), and a script that turns a PR environment's variable listing plus that manifest into the exact variable sets and removals to apply. A variable the manifest does not know about fails the run, so a new production secret can never leak into a preview unnoticed. This ticket builds and tests the logic only; wiring it into the provisioning workflow is ticket 03. Part of the first pull request (spec: preview infrastructure and lifecycle).

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] The manifest classifies every variable the API reads, using the four strategies. Image bucket write settings, scan photo bucket settings, the Admin allowlist and origins, the Gemini key and the edge secret are `remove`. Database connection settings, the development Clerk secret key and issuer, and `APP_ENV=preview` are `override`. The allowed frontend origins are `derive`. Ports, timeouts, pool settings, telemetry settings and the image host address are `inherit`. The manifest passes `yamllint`.
- [ ] Given a variable listing and a PR number, the script prints the sets (overrides and the derived allowed-origins value for that PR number) and the removals (remove-strategy variables that are present).
- [ ] The script exits non-zero and names the offending variable when the listing contains a variable absent from the manifest.
- [ ] A remove-strategy variable that is already absent produces no removal; running the script twice on its own result produces nothing to do.
- [ ] A fixture-driven check covers the cases above and runs in the repository's existing checks with no new test framework (this sets the shell-test pattern).
- [ ] Whether the pinned Railway CLI's variable delete can skip deployments is confirmed against the pinned version, and the outcome is recorded in this ticket's comments: either the script emits CLI deletes with the no-redeploy flag, or removals are emitted for Railway's API in the same batch as the sets. Setting a removed variable to empty is not used unless the API is confirmed to treat empty as unset.

## Comments

### Railway CLI removal outcome

The rejected removal options (CLI delete, `variable edit --skip-deploys`, setting to empty) and the chosen one are recorded in `docs/adr/0026-preview-variables-are-replaced-in-one-call-and-guarded-by-the-railway-environment-name.md`. Nothing was run live (no Railway credentials); ticket 05 confirms one deployment per push.

- The script prints the complete replacement set (`variables`), the `removals` and the `secrets` it left as `@@SOURCE@@` placeholders; ticket 03 sends it in one call.
- `RAILWAY_*` variables Railway provides are ignored by the script (never classified, never removed, left out of the replacement set).
- The derived value `https://cardstack-pr-{pr}.vercel.app` is the alias ticket 03 settled; the script fails when it differs from `--frontend-host`.
