# 04: Execute the exchange

**What to build:** After the cards physically change hands, each side confirms the handover. The first confirmation moves nothing; the second applies the whole Trade in one transaction, so each side loses what it offered and gains what the other offered. Received cards land in a Trade Inbox Collection the receiver files them out of later. A Trade that no longer matches what was agreed is refused whole, naming the Trade Item that went short.

This ticket also carries the one prefactor the feature needs: the quantity transfer that `Move` performs internally becomes a shared capability, with `Move` re-expressed on top of it, so a Trade deducts and adds through the same code that already handles merging into a destination and binder sources. The prefactor ships here rather than standing alone because here it gains a second caller.

**Blocked by:** 03.

**Status:** done

- [x] The first confirmation leaves the Trade `locked` and moves no quantity; the room shows it is waiting on the other side.
- [x] The second confirmation applies every Trade Item both ways inside one transaction.
- [x] Deduction takes the offered quantity from the Trade Item's source Collection first, then from the participant's other Collections holding the same Card.
- [x] When a participant's total holding of a Card is below what they offered, the whole Trade is refused, the failure names that Card, every Collection quantity stays as it was, and the Trade stays `locked`.
- [x] A Trade with no Trade Items on either side cannot be executed.
- [x] A side that receives at least one Card gets a Trade Inbox Collection named after the other participant, holding exactly the cards that arrived.
- [x] A side that receives nothing gets no Trade Inbox.
- [x] A Trade Inbox is absent from its owner's Collections list, while its cards count toward Master Inventory and Wishlist fulfilment at once.
- [x] The Trade moves to `executed`, and its sides can no longer be edited.
- [x] The same Card offered on both sides leaves each side with its original quantity and still records the Trade as executed.
- [x] `Move` keeps its current behaviour exactly: whole entries moving, merging into a destination that already holds the Card, refusing a destination that would pass its card limit, refusing a binder destination. Its existing tests stay green.
- [x] Routes tests cover the two confirmations, the short refusal, a one-sided Trade, the same Card on both sides, the empty Trade, the Trade Inbox contents and the `Move` regressions. The Trade page gains the confirm control, and one end-to-end spec walks two signed-in users from creating a Trade to cards sitting in the receiver's Trade Inbox.

## Comments

Implemented with GPT-6 Luna workers on `feat/execute-trade-exchange`. Backend commits: `b91c49a4`, `b30017a8`, `8ae23adb`. Frontend commits: `a3ff1f1d`, `81a6a782`.

Verification: the full backend script passed; frontend and Admin lint, format, typecheck, unit tests (869 frontend and 108 Admin), and builds passed. The final E2E run passed all 90 tests, including the real two-user exchange and shortage-refusal flow. The first merged E2E run failed on 18 guest pages stuck loading; a focused guest test and the full E2E rerun passed without code changes.

Review: both component reviews and the final two-axis integration review completed. Final Standards: 0 findings. Final Spec: 0 findings. Inbox filing UI and early shortage display remain in tickets 06 and 08. No push or deployment.
