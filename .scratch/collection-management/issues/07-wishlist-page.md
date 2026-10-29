# 07: Wishlist page

**What to build:** a Wishlist page at `/wishlist`, a top-level navigation item beside Inventory and Collections, where a user sees the Cards they want with how many they own of how many they want, and can change or remove items (GLOSSARY.md: Wishlist). Adding items from the catalog and by preset come in later tickets; this ticket makes the Wishlist itself exist and be usable.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] One Wishlist per user, created on first use, with items that are a Card and a wanted quantity of at least 1, unique per Card. It is separate from Inventory Entries and never counts toward Master Inventory.
- [ ] The listing returns each item with its Card, the wanted quantity and the owner's owned quantity from Master Inventory, takes the shared card-filter parameters, and adds a status filter of All, Still needed or Fulfilled. Fulfilled means owned is at least wanted and is derived on read.
- [ ] Setting a wanted quantity creates or replaces an item, removing deletes one, and Clear fulfilled removes every fulfilled item in one call. Cards in unpublished Expansion Sets are absent.
- [ ] The page reuses the Inventory list layout and the shared filter panel with the status filter, shows "owned / wanted" on every item, and shows Fulfilled as a word and not by colour alone. A changing count is announced as one status message.
- [ ] HTTP-boundary tests cover set quantity, replace, remove, Clear fulfilled, the derived status and its filter, and that Wishlist items never change Master Inventory. Screen tests cover the page, the status filter and Clear fulfilled.
