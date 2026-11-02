# 09: Wishlist Preset

**What to build:** a "Fill from sets" action on the Wishlist page that fills it once with every Card of chosen Expansion Sets that the user owns none of yet (GLOSSARY.md: Wishlist Preset). The sheet reads "I own none yet of these sets" followed by the picker.

**Blocked by:** 07 (Wishlist page).

**Status:** done

- [x] A preview call returns how many Cards a preset would add, and an apply call adds them, both using the same query: published Cards of the chosen Expansion Sets, optionally narrowed by rarity, category and tag, of which the user owns none in Master Inventory, each added with the chosen quantity.
- [x] Cards already on the Wishlist are skipped and their quantity is left alone, so applying a preset twice changes nothing. The preset is applied once and nothing about it is stored.

## Comments

- 2026-11-02: shipped in PR #74. Preview and apply live in `backend/internal/domain/service/wishlist_service.go` behind `POST/…/wishlist/preset/preview` and `/apply`; the own-none query is `FindOwnedNoneCards` in `backend/internal/domain/repository/wishlist_repository.go`; the sheet is `frontend/src/components/wishlist/preset-sheet.tsx`.
- 2026-11-02: the nine review findings on PR #74 were re-read against the code before any of them was acted on, since review text is untrusted input. The still-valid ones are in `fix(wishlist): address review findings on the preset`; the rest were already true or out of scope. One was a real bug the tests caught: `FindOwnedNoneCards` did not exclude unpublished sets.
- [x] The sheet has a multi-select of Expansion Sets shown as wrapping chips, optional narrowing filters, a quantity stepper, a live "N cards will be added" preview announced as one status message, and a confirm that applies it.
- [x] HTTP-boundary tests cover the own-none rule, skipping existing items, narrowing, unpublished sets, and that the preview count equals what is added. Screen tests cover the sheet and its live count.
