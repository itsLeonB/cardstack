# 02: Binder view

**What to build:** A user can create a binder and flip through it. From a new or empty Collection they choose a Layout (presets 2×2, 3×3, 3×4, 4×3, or custom rows and columns of 1 to 6) and a starting Leaf count (10 by default), and the Collection's page then shows the binder one Page at a time, with a grid of Slots. They swipe between Pages and can jump to a Page by number. At the Page they are viewing they can add a Leaf before or after it, or remove its Leaf, and the view shows the refusal when a Leaf cannot be removed. The view shows the card total against the Slot capacity, and the capacity field is not offered on a binder. Slots render as empty, holding a Card (image, name, number) or Unidentified, and Unidentified Slots are counted. Rows and columns can be edited, and the Layout removed, while every Slot is empty.

Spec: `.scratch/binder-scanning/spec.md` (Binder view, Layout and binder lifecycle). Built against the endpoints from ticket 01.

**Blocked by:** 01 (binder foundation)

**Status:** resolved

- [x] The Layout form offers the four presets and custom rows and columns within 1 to 6, with a Leaf count defaulting to 10 and limited to 100, and is offered only on a new or empty Collection
- [x] The Collection page of a binder shows one Page at a time, with swipe and jump-to-Page, on phone and desktop widths
- [x] The Page grid has the Layout's rows and columns, and shows empty, Card and Unidentified Slots distinctly, with the Unidentified count and the card total against capacity
- [x] Add Leaf before or after and Remove Leaf act on the Leaf of the Page being viewed, and the removal refusal is shown with the named Page
- [x] Rows, columns and Layout removal are available only while every Slot is empty
- [x] A plain Collection's page is unchanged
- [x] Covered by frontend feature tests with the generated API client mocked (create, swipe, jump, add and remove a Leaf, refusal)

## Answer

Implemented on `feat/binder-scanning` (frontend). The Layout form ("Make this a binder") lives on the Collection page of an empty plain Collection; presets are rows × columns. The current Page is component state, not a URL param. On a binder the Add cards and Scan links and the quantity controls are hidden (the backend refuses those writes), and the edit form hides Max card count. `frontend/orval.config.ts` marks `CollectionSummary.binder` nullable as a workaround until the backend tags the field nullable in `openapi.json`; remove it then. Playwright is not covered here (ticket 06 carries the binder e2e spec). Swipe is tested with synthetic touch events only.
