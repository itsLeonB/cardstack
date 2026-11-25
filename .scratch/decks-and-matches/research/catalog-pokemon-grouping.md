# Catalog Pokémon grouping

Checked on 2026-11-25. Parent completed focused Serena reads after the worker's broad search failed to establish the code facts.

## Verified code facts

- `Card`, `backend/internal/domain/entity/card.go`, lines 27–52: a Card has one `Name` string, `Category`, `Tags`, an `ExpansionSetID`, an `ImageKey`, and generic `Attributes`. This struct has no explicit canonical Pokémon/species identifier or separate locale-name fields. The generic attributes may carry more source data; their complete contents were not checked.
- `CardSummary`, `backend/internal/domain/dto/catalog_dto.go`, lines 53–63: catalog results include Card ID, name, category, tags, image URL and Expansion Set. The picker can show names and printed images without creating a new species catalog.
- `CardFilter`, the same DTO file, lines 69–89: existing search inputs include name and categories as well as set, rarity, tags and Card IDs. This establishes available filter data, not a verified new selector implementation.
- `GLOSSARY.md`: Cards are print-specific. Reprints share both name and art, so Reprint grouping cannot group every different-art printing of one Pokémon. The glossary also states that special markers such as ex and V appear in display names rather than a complete structured identity system.

## Product choices

The existing full `Name` is one small possible grouping key if the user agrees that the complete printed name defines a Key Pokémon. Different art with the same full name would group; `Dragapult` and `Dragapult ex` would remain distinct. This rule identifies a name, not a species or a unique set of attacks. It needs no inferred suffix removal.

Grouping all forms under a base species is a different requirement. It needs a rule or mapping for suffixes, regional forms, owner names and other modifiers; stripping arbitrary words would not establish reliable identity. No complete canonical mapping was verified here.

Choose separately how the default Deck name displays the grouped names and which selected print supplies an image. The user's optional name override can provide a shorter Deck title without changing identity.

## Limits

No database data, frontend selector, ingestion attributes or current services were exhaustively inspected. No services were launched and no tests ran. These findings establish the named entity and DTO fields only; they do not assert that every stored name is clean or that no additional metadata exists elsewhere.
