# Decks and match records

Status: design interview in progress; not approved for implementation.

## Confirmed scope

- First release: Pokémon TCG, Indonesian Standard play.
- Deck management starts with archetype labels only, without card lists or changes to owned quantities.
- Record every individual game in a match, not only the overall match result.
- Future deck lists must allow Proxies, which are not catalog Cards. Proxy support is outside this first release.

## Research

See [archetype research](research/archetypes.md). Archetype names group builds; they do not uniquely identify a player's deck or an exact card list. The research uses official Pokémon tournament resources and first-party Limitless pages; it does not establish a universal naming system.

## Open decisions

- Whether the user manages archetypes directly or named personal Decks that each carry an archetype.
- Who creates and maintains archetype labels, and how to handle custom or unknown labels.
- Supported match structures, game outcomes, incomplete matches and how to determine match results.
- Opponent details, match metadata, record ownership and visibility.
- Editing and deleting records, historical labels and desired summaries.

## Design constraint

The existing glossary uses a deck as an example of a physical Collection. The new match-tracking concept must not change owned quantities or treat a Proxy as an owned Card.

## Approval

Do not implement until the interview resolves the open decisions and the user confirms the design.
