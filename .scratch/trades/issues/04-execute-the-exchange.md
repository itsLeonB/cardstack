# 04: Execute the exchange

**What to build:** After the cards physically change hands, each side confirms the handover. The first confirmation moves nothing; the second applies the whole Trade in one transaction, so each side loses what it offered and gains what the other offered. Received cards land in a Trade Inbox Collection the receiver files them out of later. A Trade that no longer matches what was agreed is refused whole, naming the Trade Item that went short.

This ticket also carries the one prefactor the feature needs: the quantity transfer that `Move` performs internally becomes a shared capability, with `Move` re-expressed on top of it, so a Trade deducts and adds through the same code that already handles merging into a destination and binder sources. The prefactor ships here rather than standing alone because here it gains a second caller.

**Blocked by:** 03.

**Status:** ready-for-agent

- [ ] The first confirmation leaves the Trade `locked` and moves no quantity; the room shows it is waiting on the other side.
- [ ] The second confirmation applies every Trade Item both ways inside one transaction.
- [ ] Deduction takes the offered quantity from the Trade Item's source Collection first, then from the participant's other Collections holding the same Card.
- [ ] When a participant's total holding of a Card is below what they offered, the whole Trade is refused, the failure names that Card, every Collection quantity stays as it was, and the Trade stays `locked`.
- [ ] A Trade with no Trade Items on either side cannot be executed.
- [ ] A side that receives at least one Card gets a Trade Inbox Collection named after the other participant, holding exactly the cards that arrived.
- [ ] A side that receives nothing gets no Trade Inbox.
- [ ] A Trade Inbox is absent from its owner's Collections list, while its cards count toward Master Inventory and Wishlist fulfilment at once.
- [ ] The Trade moves to `executed`, and its sides can no longer be edited.
- [ ] The same Card offered on both sides leaves each side with its original quantity and still records the Trade as executed.
- [ ] `Move` keeps its current behaviour exactly: whole entries moving, merging into a destination that already holds the Card, refusing a destination that would pass its card limit, refusing a binder destination. Its existing tests stay green.
- [ ] Routes tests cover the two confirmations, the short refusal, a one-sided Trade, the same Card on both sides, the empty Trade, the Trade Inbox contents and the `Move` regressions. The Trade page gains the confirm control, and one end-to-end spec walks two signed-in users from creating a Trade to cards sitting in the receiver's Trade Inbox.
