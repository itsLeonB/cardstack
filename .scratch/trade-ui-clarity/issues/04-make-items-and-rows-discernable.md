# 04: Make items and rows discernable

**What to build:** A participant can pick out one Trade Item from another and one Trade from another at a glance. Each item on both sides and each row in the Trades list is its own card with the details needed to tell it apart, and the Remove action is short on screen but still names the card for assistive technology.

**Blocked by:** 01 (the final labels and title), 03 (the final Discard and Cancel actions).

**Status:** ready-for-agent

- [ ] Each Trade Item on both sides is rendered as one card showing the Card image and name, set, rarity, quantity and source Collection.
- [ ] A Trade Item that has gone short is marked with the available quantity, as the current read already reports it.
- [ ] Remove shows only the word "Remove" on screen, while its accessible name includes the card name so each button is distinguishable in a screen reader's button list.
- [ ] Remove stays on the caller's own side, works only while the Trade is `negotiating`, and is fully keyboard operable with a visible focus ring and an adequate target size.
- [ ] Each Trades list row is one card showing the name derived by ticket 01, the state and the age; cancelled Trades remain non-links in history.
- [ ] The list keeps its newest-first order, empty state, error state and Retry action, and the Discard and Cancel actions from ticket 03 sit in the card's action area apart from ordinary actions.
- [ ] Cards contain no nested interactive controls, and their content keeps a meaningful reading order.
- [ ] Presentation is checked on phone and desktop widths, in both themes, with keyboard navigation and the existing automated accessibility check.
- [ ] Focused checks and the full frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.
