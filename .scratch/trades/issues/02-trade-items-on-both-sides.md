# 02: Trade Items on both sides

**What to build:** While a Trade is negotiating, each participant adds Trade Items to their own side — a Card, a quantity, and the source Collection the copies will leave — and removes ones they no longer want. Each side sees the other's Trade Items with the Card's set and rarity, so the deal can be checked against the physical cards.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] Adding a Trade Item stores the Card, the quantity and the source Collection, on the caller's own side only.
- [ ] A quantity of zero or less is refused.
- [ ] The same Card may sit twice on one side when the two Trade Items name different source Collections; the same Card and the same source Collection twice is refused.
- [ ] Naming a Collection the caller does not own is refused.
- [ ] A source Collection holding fewer copies than the offered quantity is accepted while negotiating, because quantity is checked at execution.
- [ ] Reading the Trade returns each side's Trade Items with the Card's name, set, rarity and image.
- [ ] Adding and removing are refused once the Trade is no longer `negotiating`.
- [ ] A Trade with an empty side is allowed; only both sides empty is refused, and that refusal lands at execution.
- [ ] Routes tests cover adding, the duplicate rules, ownership, the post-lock refusal and the read. The Trade page adds and removes Trade Items and shows the other side's list.
