# 11: Guest lock UI

**Parent:** `.scratch/security-hardening/spec.md`

**What to build:** The catalog UI mirrors the API's guest lock. A Guest sees locked controls that explain themselves and a clear prompt where more results would load, and signing in from any prompt brings them back to the same catalog view. A signed-in user sees no change.

**Blocked by:** 08, 10.

**Status:** ready-for-agent

- [ ] For a Guest, the rarity, category and tag filter controls are visible but disabled with a "Sign in to use filters" prompt. Name search and Expansion Set browsing work normally.
- [ ] For a Guest on an infinite grid (catalog search and Expansion Set page), the end of the first page shows "Sign in to see more" in place of the next-page load, and no request for page 2 is made.
- [ ] For a Guest, Add to collection and quantity controls show a sign-in prompt instead of acting.
- [ ] Any `login_required` response from the API is handled gracefully as a sign-in prompt, never as a generic error toast.
- [ ] Signing in from any prompt returns the user to the same catalog view with their search intact (using the same-origin redirect rule).
- [ ] The grid's accessibility behavior from the earlier infinite-scroll tickets still holds for the Guest prompt (keyboard reachable, announced).
- [ ] Feature tests (generated client and Clerk hooks mocked) cover Guest and signed-in rendering and the return-after-sign-in behavior; frontend build, type check, lint and tests pass. Browser-level behavior is verified manually or by e2e, and the ticket says which.

## Comments
