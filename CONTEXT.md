# Cardstack

A TCG collection tracker: users record which physical cards they own, across multiple named groupings, and see a combined total across everything they own.

## Language

**Game**:
A distinct trading card game (Pokémon TCG, Riftbound, ...). Each Game owns its own set of card attributes — Pokémon's HP/types/attacks don't apply to Riftbound, and the model must not assume they're universal.

**Series**:
An optional grouping of one or more Expansion Sets released under one named product line within a Game — Pokémon TCG's Indonesian releases group into four: Scarlet & Violet, Sword & Shield ("Pedang & Perisai"), Sun & Moon ("Matahari & Bulan"), and Mega Evolution ("Evolusi Mega"). Optional because the concept doesn't apply to every Game — a Game with no Series data simply has Expansion Sets belonging to none, not a special case to work around. Carries the same regional-scoping invariant as Expansion Set: a Series is not assumed shared across regions (another region could split the same real-world card eras into a different number of Series, the same way Expansion Sets already split unevenly).
_Avoid_: confusing a Series with an Expansion Set — "Scarlet & Violet" names the Series, `SV1V` (or similar) names one Expansion Set within it.

**Expansion Set**:
A released batch of cards within a Game, identified by its own set code and scoped to one specific print edition/region — e.g. `MEG` (English/International Mega Evolution), `m1l`/`m1s` (Japan's Mega Brave / Mega Symphonia), `MA1` (Indonesian Evolusi Mega). Belongs to zero or one Series. Regions are not locale variants of one shared set: the same real-world card era can be split into a different number of Expansion Sets per region (Japan splits Mega Evolution into two sets where Indonesia and English each have one). MVP models only Indonesian print editions, across the four Series listed above; other regions (EN, JP) are out of scope until explicitly added, each as their own distinct Expansion Sets.
_Avoid_: assuming an Expansion Set is shared or reused across regions by default.

**Card**:
A specific printed card design within exactly one Expansion Set — number and rarity are scoped to that Expansion Set, not shared across regions. Where one upstream source bundles several locale names onto a single record (TCGDex's Asia-region SV data carries `id`/`ja`/`zh-tw`/`th` names on one shared file, because those storefronts happen to sell an identical print run), that's a detail of that source's data shape for that specific print edition — not a general rule that a Card spans regions. A future Japan or English Expansion Set gets its own, unrelated Cards, even for "the same" real-world card. This is the unit quantities are tracked against in MVP (see Card Variant).

**Card Category**:
The primary, mutually-exclusive structural classification every Card has within its Game — Pokémon TCG's are `Pokémon`/`Trainer`/`Energy`; another Game defines its own vocabulary (e.g. a hypothetical Creature/Land/Spell split). Exactly one Category per Card. Generalizes across Games as a concept even though the vocabulary of values is Game-specific — distinct from Card Tag, which is non-exclusive and secondary.

**Card Rarity**:
The print-rarity tier of a Card, exactly one per Card. A first-class, per-Game lookup table (not plain text) — Pokémon TCG's vocabulary (`C`/`U`/`R`/.../`SAR`/`MUR`/`FUR`/...) is a cryptic code, unlike Card Category's already-legible labels, so it's decoded through a real row rather than stored raw (ADR-0009). Scoped to `(Game, code)`, not to Series or Expansion Set: within Pokémon TCG the *active* tier roster rotates print-era to print-era (older Series produced `UR`; Evolusi Mega instead produces `SAR`/`MUR`/`FUR`; `MUR` itself stops appearing after Evolusi Mega's fifth set) — but no code has been observed to change meaning, so a Game-wide vocabulary is correct even though not every Expansion Set uses every row in it.
_Avoid_: assuming an unused rarity code on an older/newer Expansion Set means the vocabulary is Set-scoped — it means that tier simply wasn't produced in that print run, not that the code means something else there.

**Card Tag**:
Zero or more secondary classification labels a Card carries beyond its Category — Pokémon TCG's Trainer/Energy subtype markers (`Item`/`Supporter`/`Stadium`/`Pokémon Tool`/`Energi Dasar`/`Energi Spesial`) are what's actually modeled here, one structurally-exposed value per Card. Vocabulary is entirely Game-defined; what generalizes across Games is the mechanism (an arbitrary, multi-valued label set per Card), not the labels themselves.
_Avoid_: conflating Card Tag with Card Variant — a Tag describes what a card *is* (its kind/subtype), a Variant would describe how a specific physical copy was *printed*. Also avoid assuming Pokémon's own special/promotional markers (`ex`/`GX`/`V`/`VMAX`/...) live here in MVP — the source doesn't expose them as structured data (only baked into the card's display name), so they're captured best-effort into Card's Game-specific `attributes` instead, not as a Tag, until a source exposes them cleanly enough to filter on.

**Card Variant**:
Not modeled in MVP. A specific print finish of a Card (normal, reverse holo, holo, first edition, ...) — dropped from the schema because no data source currently ingested supplies finish/variant data. Inventory Entry and Master Inventory track against Card directly until a source that carries finish data is actually added. Do not conflate with Card Tag (see above).

**Collection**:
A user-owned, named grouping of Cards with quantities (a binder, box, or deck). Has a user-given title, an optional description, and an optional maximum card-count limit. May carry an informal type label (binder/box/deck) for the owner's own organization; MVP applies no type-specific rules (no deck-size enforcement, no binder page/slot layout — that's a later feature). Strictly private to its owner in MVP; sharing is a post-MVP concern.

**Inventory Entry**:
The atomic record of how many of one Card a user holds within one Collection. This is where quantity actually lives — Collections and Master Inventory are both just views over Inventory Entries.
_Avoid_: "Inventory" alone — ambiguous between this (atomic, per-Collection) and Master Inventory (aggregate, per-user). Always say which one.

**Master Inventory**:
Not a stored entity. The aggregate of a user's Inventory Entries across *all* their Collections, grouped by Card — "how many of this card do I own in total, anywhere." Computed on read, not written to.

**Wishlist**:
Not modeled in MVP. A future concept for cards a user wants but doesn't own; do not conflate with Inventory Entry (owned quantity) until it's actually built.
