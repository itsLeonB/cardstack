# 06: Frontend image variants

**Parent:** `.scratch/security-hardening/spec.md` (see ADR-0016)

**What to build:** Card tiles, the card detail view and Expansion Set covers request appropriately sized, modern-format images from our image host instead of full-size originals. A phone loads small images, a high-density screen gets sharper ones, and a missing image shows the existing name placeholder.

**Blocked by:** 05. Real-world verification also needs ticket 02.

**Status:** ready-for-agent

- [ ] A single image helper turns the API's image address into a sized address and a responsive source set: card tiles at 240 and 480 wide, the detail view at 720, Expansion Set covers at 128, with automatic format selection and a redirect to the original when the transformation allowance is exhausted.
- [ ] Tiles, the detail view and Expansion Set covers use the helper and declare dimensions so layout does not shift.
- [ ] An empty address, or an address not on the configured image host (local development, hosting not configured), is returned unchanged; an empty address shows the existing placeholder.
- [ ] The placeholder still appears when an image fails to load.
- [ ] Feature tests (generated client mocked) cover the sized, empty, off-host and load-error cases; the image-host setting is added to the frontend env example and the deployment doc.
- [ ] Frontend build, type check, lint and tests pass. UI behavior in a real browser is verified manually or by the existing e2e layer, and the ticket says which.

## Comments
