# 02: Add a Trade Item from a picker page

**What to build:** A participant adding a Card to their side of a Trade gets a page of its own. It uses the catalog search they already know, asks for the source Collection and quantity there, and returns them to the Trade detail page with the new Trade Item visible. The detail page no longer carries an inline add form.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] An add action on the Trade detail page opens a dedicated picker route for that Trade.
- [ ] The picker reuses the existing catalog search results and its filters, pagination and loading behaviour; no second search implementation is introduced.
- [ ] The source Collection and quantity are entered on the picker page and stored on the Trade Item exactly as the inline form stored them.
- [ ] Submitting adds the Trade Item through the existing add operation, returns to the Trade detail page, and shows the new item there without a manual refresh.
- [ ] A zero or negative quantity, a Collection the caller does not own, a duplicate Card and Collection pair, or a Trade no longer `negotiating` is refused with a plain message that names the cause.
- [ ] The same Card can be offered twice from two different Collections, as the domain requires.
- [ ] Browser back from the picker returns to the Trade detail page and adds nothing.
- [ ] The inline add form is removed from the detail page, and adding and removing remain limited to the caller's own side while the Trade is `negotiating`.
- [ ] The picker shows only what the caller may offer and never exposes another user's holdings; Guest, non-participant and not-found behaviour is unchanged.
- [ ] Authenticated end-to-end coverage proves the round trip from detail to picker and back, and keeps the existing access, cancel and execute assertions intact.
- [ ] Focused checks and the full frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.
