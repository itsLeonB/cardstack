# 09: Wishlist Preset

**What to build:** a "Fill from sets" action on the Wishlist page that fills it once with every Card of chosen Expansion Sets that the user owns none of yet (GLOSSARY.md: Wishlist Preset). The sheet reads "I own none yet of these sets" followed by the picker.

**Blocked by:** 07 (Wishlist page).

**Status:** ready-for-agent

- [ ] A preview call returns how many Cards a preset would add, and an apply call adds them, both using the same query: published Cards of the chosen Expansion Sets, optionally narrowed by rarity, category and tag, of which the user owns none in Master Inventory, each added with the chosen quantity.
- [ ] Cards already on the Wishlist are skipped and their quantity is left alone, so applying a preset twice changes nothing. The preset is applied once and nothing about it is stored.
- [ ] The sheet has a multi-select of Expansion Sets shown as wrapping chips, optional narrowing filters, a quantity stepper, a live "N cards will be added" preview announced as one status message, and a confirm that applies it.
- [ ] HTTP-boundary tests cover the own-none rule, skipping existing items, narrowing, unpublished sets, and that the preview count equals what is added. Screen tests cover the sheet and its live count.
