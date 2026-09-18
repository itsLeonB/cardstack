# Cardstack

A TCG collection tracker: users record which physical cards they own, across multiple named groupings, and see a combined total across everything they own.

## Language

**Game**:
A distinct trading card game (Pokémon TCG, Riftbound, ...). Each Game owns its own set of card attributes — Pokémon's HP/types/attacks don't apply to Riftbound, and the model must not assume they're universal.

**Expansion Set**:
A released batch of cards within a Game, identified by its own set code and scoped to one specific print edition/region — e.g. `MEG` (English/International Mega Evolution), `m1l`/`m1s` (Japan's Mega Brave / Mega Symphonia), `MA1` (Indonesian Evolusi Mega). Regions are not locale variants of one shared set: the same real-world card era can be split into a different number of Expansion Sets per region (Japan splits Mega Evolution into two sets where Indonesia and English each have one). MVP models only Indonesian print editions (SV, MA); other regions (EN, JP) are out of scope until explicitly added, each as their own distinct Expansion Sets.
_Avoid_: assuming an Expansion Set is shared or reused across regions by default.

**Card**:
A specific printed card design within exactly one Expansion Set — number and rarity are scoped to that Expansion Set, not shared across regions. Where one upstream source bundles several locale names onto a single record (TCGDex's Asia-region SV data carries `id`/`ja`/`zh-tw`/`th` names on one shared file, because those storefronts happen to sell an identical print run), that's a detail of that source's data shape for that specific print edition — not a general rule that a Card spans regions. A future Japan or English Expansion Set gets its own, unrelated Cards, even for "the same" real-world card.

**Card Variant**:
A specific print finish of a Card (normal, reverse holo, holo, first edition, ...). This — not the Card itself — is the unit that quantities are tracked against, since a normal print and a reverse holo of the same Card are different objects to a collector.

**Collection**:
A user-owned, named grouping of Card Variants with quantities (a binder, box, or deck). Has a user-given title, an optional description, and an optional maximum card-count limit. May carry an informal type label (binder/box/deck) for the owner's own organization; MVP applies no type-specific rules (no deck-size enforcement, no binder page/slot layout — that's a later feature). Strictly private to its owner in MVP; sharing is a post-MVP concern.

**Inventory Entry**:
The atomic record of how many of one Card Variant a user holds within one Collection. This is where quantity actually lives — Collections and Master Inventory are both just views over Inventory Entries.
_Avoid_: "Inventory" alone — ambiguous between this (atomic, per-Collection) and Master Inventory (aggregate, per-user). Always say which one.

**Master Inventory**:
Not a stored entity. The aggregate of a user's Inventory Entries across *all* their Collections, grouped by Card Variant — "how many of this card do I own in total, anywhere." Computed on read, not written to.

**Wishlist**:
Not modeled in MVP. A future concept for cards a user wants but doesn't own; do not conflate with Inventory Entry (owned quantity) until it's actually built.
