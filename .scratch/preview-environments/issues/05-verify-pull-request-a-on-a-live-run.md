# 05: Verify the preview infrastructure on a live PR run

**What to build:** Proof that the first pull request works end to end on a real pull request, since the workflow cannot be tested locally and workflow changes in a PR take effect for that PR. This needs the repository's Neon, Railway and Vercel credentials, so it is done by the maintainer.

**Blocked by:** 02, 03, 04.

**Status:** ready-for-human

- [ ] Two consecutive pushes to the PR each produce exactly one API deployment.
- [ ] Data created in the preview after the first push is still there after the second push (the Neon branch is kept).
- [ ] The PR environment has none of the remove-strategy variables and has `APP_ENV=preview`.
- [ ] The frontend alias resolves, and a development Clerk sign-in succeeds against the API (the allowed origin matches).
- [ ] The Railway domain existed before the API's first deployment; if it did not, record how the workflow had to change.
- [ ] A PR environment whose provisioning run is skipped or cancelled refuses to boot with the guard's message instead of running against production.
- [ ] Closing the PR deletes the Neon branch, the alias, the deployments and the branch-scoped frontend variables.
- [ ] Any workflow fix needed is committed on the same pull request, and the outcome is recorded in this ticket's comments.
