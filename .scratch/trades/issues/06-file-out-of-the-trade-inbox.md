# 06: File out of the Trade Inbox

**What to build:** Received cards wait in a Trade Inbox until the receiver files them into a Collection of their own. Filing takes a chosen quantity of a Card from the inbox into a chosen Collection; the inbox disappears once its last card has been filed, so no empty leftovers accumulate among the owner's Collections.

**Blocked by:** 04.

**Status:** ready-for-agent

- [ ] Filing moves a chosen quantity of a Card from the Trade Inbox into a Collection the caller owns.
- [ ] Filing into a Collection whose card limit would be passed is refused whole, and nothing moves.
- [ ] Filing into another user's Collection, or out of another user's Trade Inbox, is refused.
- [ ] The Trade Inbox is deleted when its last Inventory Entry leaves, and survives while anything remains.
- [ ] The Trade Inbox refuses rename, quantity edit and direct delete; filing is the only way cards leave it.
- [ ] The inbox page lists what arrived with the Card's set and rarity, and offers filing.
- [ ] Routes tests cover filing, the capacity refusal, the disappearance, and each refused operation. The end-to-end spec extends past execution to filing a card and seeing the inbox go.
