# 05: Make binder setup optional

**What to build:** Let an owner start an empty Collection by adding or scanning Cards, and choose “Make this a binder” only when they want binder setup. Make Scan cards and Add Cards easier to identify with icons.

**Blocked by:** None (can start immediately after implementation approval).

**Status:** claimed

**Execution:** approved by the user's implement-spec request; integration branch `feat/mobile-ui-ux`.

## Acceptance criteria

### Collection and binder flow

- [ ] Remove the default inline binder setup from an eligible empty Collection. Show a secondary “Make this a binder” action only when the owner’s Collection has no Layout and zero Cards.
- [ ] An explicit click opens the existing binder creation form in the existing named Dialog Module. Never auto-open it on page load, first visit, or an empty state.
- [ ] Cancel or dismiss without creating a Layout. Preserve the existing submission/pending behavior and do not introduce a write merely by opening the dialog.
- [ ] Preserve Layout choices and validation: default 3-by-3 Slots per Page and 10 Leaves; rows/columns remain 1–6 and Leaves remain 1–100. Keep inline errors and the existing creation contract.
- [ ] A successful creation closes the dialog and shows the updated binder using existing invalidation/read-back behavior without a manual refresh.

### Actions and invariants

- [ ] Add a decorative camera icon beside Scan cards and a plus/Card icon beside Add Cards using the existing Remix icons and button/link patterns.
- [ ] Retain both visible action labels, destinations, and eligibility. Scan remains available for binders; Add Cards does not gain binder availability.
- [ ] Preserve ADR-0020: only an empty Collection can gain a Layout, Slots determine binder quantities, and the existing capacity-rule change still applies. Do not introduce bulk Inventory Entry quantity writes for binders.
- [ ] Preserve the current page structure, brand, and both themes. Keep the dialog named, keyboard-operable, focus-trapped when modal, dismissible with Escape, and able to return focus to its trigger.
- [ ] Changed mobile actions target at least 44-by-44 CSS pixels and remain usable at text zoom without unwanted overflow.

## Verification

- [ ] Adapt the existing Collection-page and binder-creation tests to open setup by clicking the action. Check initial closure, eligibility, cancel with no write, success transition, and an inline failure. Reuse existing Layout-limit tests rather than repeating every validation case through the dialog.
- [ ] Keep ScanLink and AddCardsLink destination tests, and assert their accessible text remains. Do not assert decorative SVG structure or icon classes.
- [ ] Exercise keyboard dismissal/focus return and rendered accessibility in both themes. Run the relevant compose frontend verification checks; this flow must also work before mobile bottom navigation lands.

## Scope limits

No populated-Collection conversion, new binder type rules, new Layout model, automatic setup prompt, scanning redesign, or dependency on another implementation ticket. Read the feature’s approved design spec before implementation.
