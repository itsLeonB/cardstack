# Cardstack

A TCG collection tracker: users record which physical cards they own, across multiple named groupings, and see a combined total across everything they own.

## Language

**Game**:
A distinct trading card game (Pokémon TCG, Riftbound, ...). Each Game owns its own set of card attributes — Pokémon's HP/types/attacks don't apply to Riftbound, and the model must not assume they're universal.

**Series**:
An optional grouping of one or more Expansion Sets released under one named product line within a Game — Pokémon TCG's Indonesian releases group into four: Scarlet & Violet, Sword & Shield ("Pedang & Perisai"), Sun & Moon ("Matahari & Bulan"), and Mega Evolution ("Evolusi Mega"). Optional because the concept doesn't apply to every Game — a Game with no Series data simply has Expansion Sets belonging to none, not a special case to work around. Carries the same regional-scoping invariant as Expansion Set: a Series is not assumed shared across regions (another region could split the same real-world card eras into a different number of Series, the same way Expansion Sets already split unevenly). A Series has no release date of its own — the date used to order Series for display is derived as the earliest ReleaseDate among its Expansion Sets, computed at read time rather than stored (the attribute ADR-0008 anticipated Series might eventually carry).
_Avoid_: confusing a Series with an Expansion Set — "Scarlet & Violet" names the Series, `SV1V` (or similar) names one Expansion Set within it.

**Expansion Set**:
A released batch of cards within a Game, identified by its own set code and scoped to one specific print edition/region — e.g. `MEG` (English/International Mega Evolution), `m1l`/`m1s` (Japan's Mega Brave / Mega Symphonia), `MA1` (Indonesian Evolusi Mega). Belongs to zero or one Series. Regions are not locale variants of one shared set: the same real-world card era can be split into a different number of Expansion Sets per region (Japan splits Mega Evolution into two sets where Indonesia and English each have one). MVP models only Indonesian print editions, across the four Series listed above; other regions (EN, JP) are out of scope until explicitly added, each as their own distinct Expansion Sets.
_Avoid_: assuming an Expansion Set is shared or reused across regions by default.

**Card**:
A specific printed card design within exactly one Expansion Set — number and rarity are scoped to that Expansion Set, not shared across regions. Where one upstream source bundles several locale names onto a single record (TCGDex's Asia-region SV data carries `id`/`ja`/`zh-tw`/`th` names on one shared file, because those storefronts happen to sell an identical print run), that's a detail of that source's data shape for that specific print edition — not a general rule that a Card spans regions. A future Japan or English Expansion Set gets its own, unrelated Cards, even for "the same" real-world card. This is the unit quantities are tracked against in MVP (see Card Variant).

**Card Embedding**:
The vector that an image-embedding model makes from a Card's hosted image, stored with the name of the model that produced it and a source naming which image was embedded, so a Card can hold several (one per model and source). Card scanning matches a photo to a Card by comparing the photo's vector with Card Embeddings of the same model; a Card with no hosted image has none and cannot be matched.
_Avoid_: calling it a Card attribute or a Card Variant — it describes the picture, not the print, and changing the model replaces every row.

**Card Category**:
The primary, mutually-exclusive structural classification every Card has within its Game — Pokémon TCG's are `Pokémon`/`Trainer`/`Energy`; another Game defines its own vocabulary (e.g. a hypothetical Creature/Land/Spell split). Exactly one Category per Card. Generalizes across Games as a concept even though the vocabulary of values is Game-specific — distinct from Card Tag, which is non-exclusive and secondary.

**Card Rarity**:
The print-rarity tier of a Card, exactly one per Card. A first-class, per-Game lookup table (not plain text) — Pokémon TCG's vocabulary (`C`/`U`/`R`/.../`SAR`/`MUR`/`FUR`/...) is a cryptic code, unlike Card Category's already-legible labels, so it's decoded through a real row rather than stored raw (ADR-0009). Scoped to `(Game, code)`, not to Series or Expansion Set: within Pokémon TCG the *active* tier roster rotates print-era to print-era (older Series produced `UR`; Evolusi Mega instead produces `SAR`/`MUR`/`FUR`; `MUR` itself stops appearing after Evolusi Mega's fifth set) — but no code has been observed to change meaning, so a Game-wide vocabulary is correct even though not every Expansion Set uses every row in it.
_Avoid_: assuming an unused rarity code on an older/newer Expansion Set means the vocabulary is Set-scoped — it means that tier simply wasn't produced in that print run, not that the code means something else there.

**Regulation Mark**:
A letter (`A`-`J` observed so far) printed on every Card, controlling official tournament format legality (Standard/Expanded) — not a rarity, not a category, and not decoded into a lookup row. Captured verbatim into Card's Game-specific `attributes` rather than resolved into a format label: which marks are Standard-legal is a floor comparison that moves every yearly rotation, so freezing a computed "Standard"/"Expanded" bucket at ingestion time would go stale (ADR-0010). `A`/`B`/`C` are legitimate, Asia-only marks from the Sun & Moon era (the "Matahari & Bulan" Series) — expected to appear on cards from that Series, not a scraping error.
_Avoid_: confusing this with Card Rarity — both are short, cryptic-looking per-card codes, but Rarity is print-tier (`C`/`U`/`R`/`RR`/...) and Regulation Mark is format legality (`A`-`J`); the two vocabularies even coincidentally overlap on the letter `C`, which means something completely different in each. Also avoid inventing a third "format" bucket beyond Standard/Expanded on this data — no such official format exists (ADR-0010).

**Card Tag**:
Zero or more secondary classification labels a Card carries beyond its Category — Pokémon TCG's Trainer/Energy subtype markers (`Item`/`Supporter`/`Stadium`/`Pokémon Tool`/`Energi Dasar`/`Energi Spesial`) are what's actually modeled here, one structurally-exposed value per Card. Vocabulary is entirely Game-defined; what generalizes across Games is the mechanism (an arbitrary, multi-valued label set per Card), not the labels themselves.
_Avoid_: conflating Card Tag with Card Variant — a Tag describes what a card *is* (its kind/subtype), a Variant would describe how a specific physical copy was *printed*. Also avoid assuming Pokémon's own special/promotional markers (`ex`/`GX`/`V`/`VMAX`/...) live here in MVP — the source doesn't expose them as structured data (only baked into the card's display name), so they're captured best-effort into Card's Game-specific `attributes` instead, not as a Tag, until a source exposes them cleanly enough to filter on.

**Card Variant**:
Not modeled in MVP. A specific print finish of a Card (normal, reverse holo, holo, first edition, ...) — dropped from the schema because no data source currently ingested supplies finish/variant data. Inventory Entry and Master Inventory track against Card directly until a source that carries finish data is actually added. Do not conflate with Card Tag (see above).

**Source**:
Who writes Card and Expansion Set rows: the Pokémon Asia site, TCG Collector, or an Admin. Every Card and Expansion Set records two Sources: the one that created it and the one that last updated it. A Source changes a row only while it is also the one that last updated it, so no Source overwrites another's: whichever wrote a row keeps it, even when a later Source publishes the same Expansion Set or Card, and a Source adds what is missing and leaves the rest alone. An Admin's edit makes the Admin the last updater, which hands the row to the Admin from then on; the ingesters skip it. Only a manual change by the operator hands a row back. Sources agree on identity by Expansion Set code and by the Card's number within its Expansion Set, never by name (TCG Collector's set names are English translations); numeric Card numbers compare as numbers, so `1` and `001` are the same Card. One exception: TCG Collector may fill the logo of any Expansion Set whatever its Sources, but only while that Expansion Set has none, and it changes nothing else on the row, including who last updated it. An Expansion Set shows its logo when it has one and falls back to the cover from the Pokémon Asia site otherwise.
_Avoid_: treating the Pokémon Asia site as always winning — it is the main Source in that it is run first and fills most of the catalog, but a row another Source wrote is not replaced by it.

**Published**:
Whether an Expansion Set, with all its Cards, is visible to anyone but an Admin. An unpublished Expansion Set is in the catalog but absent from every user-facing view: browsing, search, filters, counts, inventory pickers and card scanning. Existing Expansion Sets are published. A set that TCG Collector adds because no other Source had its code starts unpublished, so an operator can check its Cards before anyone can record holdings against them; the operator publishes it by hand. A Card has no Published state of its own: a Card added to an already published Expansion Set is visible at once.
_Avoid_: using "published" for a Card, or for Series — it is a property of the Expansion Set only.

**Curated**:
A Card or Expansion Set that an Admin last updated. Exists because some real Expansion Sets have no usable upstream source — the Indonesian promo set `SM-P` ("Kartu Promo Matahari & Bulan") is absent from the Asia site — so their Cards must be entered by hand.
_Avoid_: calling a Card "custom" or "manual" — those say how it was typed, not whose row it is.

**Collection**:
A user-owned, named grouping of Cards with quantities (a binder, box, or deck). Has a user-given title, an optional description, and an optional maximum card-count limit (a Collection with a Layout has none: its capacity is its Slot count). May carry an informal type label (binder/box/deck) for the owner's own organization; MVP applies no type-specific rules (no deck-size enforcement). May optionally carry a Layout, which makes it a binder-style Collection: one physical binder is one Collection. Strictly private to its owner in MVP; sharing is a post-MVP concern.

**Layout**:
The physical shape of a binder-style Collection: how many rows and columns each Page has, and how many Leaves there are. Uniform across the whole Collection. A Collection without a Layout (a box, a deck) has no Leaves, Pages or Slots and tracks quantity only.

**Leaf**:
One physical binder sheet, with a front Page and a back Page. The unit a person adds to or removes from a binder, always as both faces together. Leaves are ordered in the binder, and inserting or removing one renumbers the Pages after it, the way page numbers shift in a real binder.

**Page**:
One face of a Leaf, holding a fixed grid of Slots. Pages are numbered in the order a person turns through the binder.

**Slot**:
One pocket on a Page, addressed by its Page, row and column. Holds at most one physical card and is in one of three states: empty, holding a Card, or Unidentified. In a Collection with a Layout, the quantity of a Card is the number of its Slots; two copies of a Card take two Slots.
_Avoid_: calling a Slot a quantity or an Inventory Entry — an Inventory Entry says how many, a Slot says where.

**Unidentified**:
The state of a Slot or a Scan that holds a physical card the app could not match to a Card, whether because matching failed, no candidate was good enough, or the user marked it by hand. It counts toward no Card's quantity and no Master Inventory, since it has no Card, but it is a physical card in the binder and is shown as a count. The user resolves it by placing a Card in the Slot. An Unidentified Scan has no Slot to hold it and is set aside when its Draft Addition is confirmed.
_Avoid_: "unknown card" — Phase 1 already uses that for a Card id the server does not know; here the card exists but has no identity yet.

**Inventory Entry**:
The atomic record of how many of one Card a user holds within one Collection. This is where quantity actually lives — Collections and Master Inventory are both just views over Inventory Entries.
_Avoid_: "Inventory" alone — ambiguous between this (atomic, per-Collection) and Master Inventory (aggregate, per-user). Always say which one.

**Draft Addition**:
An unsaved, transient list of Scans that a user builds by photographing cards, aimed at one chosen Collection. Nothing in it is owned yet: it becomes Inventory Entries (or, in a Collection with a Layout, copies placed in empty Slots, which keep the Inventory Entries in step) only when the user reviews it and confirms, and abandoning it leaves the Collection untouched. Quantities are additions to what the Collection already holds, not new totals, and the Scans that chose the same Card add together.
_Avoid_: "inventory" or "collection" for the draft — it is neither; it is a pending change to one Collection's Inventory Entries.

**Scan**:
One photographed card in a Draft Addition, with the Card chosen for it and a quantity to add. A Scan shows the matcher's top candidate until the user picks another; when the matcher was not confident it also keeps the other candidates it ranked, so the user can change the choice, and when the matcher was confident it keeps only the top one. A Scan is pending while its photo is still being matched. A Scan can instead be marked Empty (the photo shows no card) or Unidentified (it shows a card but none of the candidates fits); a marked Scan adds nothing and is set aside when the Draft Addition is confirmed. Photographing the same Card twice makes two Scans; they are never merged, and the user may raise a Scan's quantity by hand.
_Avoid_: "match" or "candidate" for the Scan itself — a candidate is one Card the matcher offered for a Scan, and the Scan is the row that holds the choice among them.

**Page Review**:
The result of photographing one Page of a binder-style Collection: one cell per Slot, each with the Card the matcher chose, the other candidates it ranked, and any choice the user made by hand (a different Card, Empty, or Unidentified). Aimed at exactly one Page, and nothing in it is placed in the binder until the user confirms, which replaces that whole Page. A Page Review is pending from the moment its cells are cut until it is confirmed or discarded, and the user can leave and come back to a pending one. A Page has at most one pending Page Review per user.
_Avoid_: "Scan" or "Draft Addition" for it — a Scan is one photographed card in a Draft Addition, and a Draft Addition adds to a Collection's quantities, while a Page Review rewrites the Slots of one Page.

**Detection**:
What the matcher returned for one photographed card (a Scan, or one cell of a Page Review): the ranked Cards it offered, whether it was confident in the top one, and whether it saw no card at all. It is the matcher's answer as it was given; what the user then chose is separate and may differ.
_Avoid_: using it for the choice itself — the Scan or cell holds what the user chose, the Detection holds what the matcher said.

**Correction**:
A confirmed Scan or cell whose final choice differs from its Detection: the user chose a Card other than the matcher's top one, marked it Empty (no card is there) or Unidentified (a card is there but no candidate fits), or placed a Card where the Detection was Empty or offered none. Only confirming makes a Correction: a choice changed and changed back, or one in a review that was never confirmed, is none. A confirmed Scan or cell without a Correction is a confirmation. Corrections are kept so the operator can see which Cards the matcher is least sure of and which photos it handles badly.

**Reprint**:
A Card that shares its name and its art with another Card in a different Expansion Set. The matcher cannot tell Reprints apart by the picture, so it also offers a Card's other Reprints: the printed set is the part the user has to choose.
_Avoid_: calling a card re-released with new art a Reprint — that is a different Card.

**Master Inventory**:
Not a stored entity. The aggregate of a user's Inventory Entries across *all* their Collections, grouped by Card — "how many of this card do I own in total, anywhere." Computed on read, not written to.

**Auth Identity**:
The (auth provider, auth subject) pair that proves who a caller is, issued by the identity provider (Clerk, ADR-0015), never by Cardstack. A User row is the Cardstack-side record of one Auth Identity, plus an email; the User's profile owns Collections. Cardstack stores no passwords or sessions.
_Avoid_: "account" for the Clerk-side login and "user" for the Cardstack-side row interchangeably — the identity lives at the provider, the row only points at it.

**Guest**:
A caller with no `Authorization` header. A Guest gets a deliberate catalog preview: browse Series and Expansion Sets, name search, and one page of 24 results; filters on rarity, category and tag, facet counts and further pages require an Auth Identity. A request whose token is present but invalid is not a Guest, it is rejected with 401.
_Avoid_: calling an expired-token caller a Guest.

**Admin**:
A signed-in caller the operator has named as allowed to write catalog content by hand (Expansion Sets and Cards, with their images). An Admin is an ordinary Auth Identity that the operator has listed; it is not a role stored in Cardstack, and nothing in the product shows a signed-in user that Admin exists. Whatever an Admin writes becomes Curated.
_Avoid_: "admin user" or "admin account" — there is no separate account type, only an Auth Identity that happens to be listed.

**Wishlist**:
Not modeled in MVP. A future concept for cards a user wants but doesn't own; do not conflate with Inventory Entry (owned quantity) until it's actually built.
