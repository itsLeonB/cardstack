# 06: Frontend image variants

**Parent:** `.scratch/security-hardening/spec.md` (see ADR-0016)

**What to build:** Card tiles, the card detail view and Expansion Set covers request appropriately sized, modern-format images from our image host instead of full-size originals. A phone loads small images, a high-density screen gets sharper ones, and a missing image shows the existing name placeholder.

**Blocked by:** 05. Real-world verification also needs ticket 02.

**Status:** code, tests and docs are in; real-browser verification is still open. Neither a manual browser check nor an e2e covers the UI yet: it needs ticket 02's image host with Image Transformations enabled and `VITE_IMAGE_HOST` set, to confirm actual resizing, format negotiation and the redirect fallback.

- [x] A single image helper turns the API's image address into a sized address and a responsive source set: card tiles at 240 and 480 wide, the detail view at 720, Expansion Set covers at 128, with automatic format selection and a redirect to the original when the transformation allowance is exhausted.
- [x] Tiles, the detail view and Expansion Set covers use the helper and declare dimensions so layout does not shift.
- [x] An empty address, or an address not on the configured image host (local development, hosting not configured), is returned unchanged; an empty address shows the existing placeholder.
- [x] The placeholder still appears when an image fails to load.
- [x] Feature tests (generated client mocked) cover the sized, empty, off-host and load-error cases; the image-host setting is added to the frontend env example and the deployment doc.
- [ ] Frontend build, type check, lint and tests pass (they do). UI behavior in a real browser is verified manually or by the existing e2e layer, and the ticket says which.

## Comments
