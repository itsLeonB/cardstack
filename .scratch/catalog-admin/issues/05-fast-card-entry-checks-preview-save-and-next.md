# 05: Fast card entry: file checks, preview, save and next

**What to build:** Entering a long run of cards is quick and mistakes are caught before anything is sent. The `/admin` card form checks the file, previews it, and offers "save and next", which keeps the chosen set and moves to the next collector number. The checks live in the frontend upload helper, not the backend, so the helper is reusable (ADR-0019). They are a convenience for one trusted editor, not a security boundary.

**Blocked by:** 04.

**Status:** resolved

- [ ] Before requesting a signed address, the page rejects a file that is not a JPEG, PNG or WebP, that is over the size limit, or whose dimensions are wrong, and shows why
- [ ] A Card cannot be saved without an image file selected
- [ ] The chosen image is previewed before saving
- [ ] "Save and next" keeps the chosen Expansion Set and advances the collector number, keeping the zero padding of the entered number
- [ ] The upload logic is a standalone helper separate from the page component, so another feature can reuse it
- [ ] A frontend route test (mocked API, existing route-test style) covers rejection of a wrong type, an oversize file and wrong dimensions without any request sent, and the save-and-next flow
- [ ] The docs state that R2 cannot cap a presigned upload's size, so these checks are not enforcement
