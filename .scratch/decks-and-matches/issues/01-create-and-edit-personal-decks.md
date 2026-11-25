# 01: Create and edit personal Decks

**What to build:** Let a signed-in user create, browse and edit private personal Decks for Indonesian Standard play. Select Key Pokémon from the existing catalog, without recording a card list or changing card ownership.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Create and persist a Deck with one or more Key Pokémon chosen from published catalog Pokémon Cards, without requiring the user to own them and without a maximum of two.
- [ ] Group Key Pokémon by full printed name across art and Expansion Sets. Keep suffixes, forms and owner names distinct; different-art Dragapult ex prints group together, while Dragapult and Dragapult ex remain separate. Do not use the narrower Reprint grouping.
- [ ] Use the Key Pokémon names as the default Deck name. Allow an optional name override and still show the Key Pokémon below it in smaller text. Allow several personal Decks with the same Key Pokémon, with or without distinct overrides.
- [ ] Persist edits to the owner's Deck name override and Key Pokémon. Guests and other users cannot read or change private Decks, including through direct requests. Deck operations do not change Collections, Inventory Entries or Master Inventory.
- [ ] Automated checks cover grouping and invalid selections, persistence and owner isolation, and a browser journey that creates a Deck, reloads it and edits it. Reuse existing catalog access and verification practices; do not add a species catalog, archetype field, card-list builder or inventory link.
