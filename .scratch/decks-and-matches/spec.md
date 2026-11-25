# Decks and match records

Status: design interview in progress; not approved for implementation.

## Confirmed scope

- First release: Pokémon TCG, Indonesian Standard play.
- Deck management starts with named personal Decks, without card lists or changes to owned quantities. The user confirmed separate named Decks in round 2.
- The user describes deck names as one or two main archetypes: Dragapult Blaziken or Ceruledge. The meaning of archetype versus a name's component remains open pending further research; do not assume the initial label model is accepted.
- Record every individual game in a match, not only the overall match result.
- Support best-of-one and best-of-three matches, with the match result determined from game results.
- Record the opponent's Deck (described by the user as one or two main archetypes), plus an optional person's name. No opponent Cardstack account is required.
- Allow match entry during or after play; unfinished matches can be saved and resumed.
- Each match has a date, optional event name and optional notes. Structured tournament management is a future goal, outside this release.
- Decks and match records are private to their owner in this release.
- Future deck lists must allow Proxies, which are not catalog Cards. Proxy support is outside this first release.

## Research

See [archetype research](research/archetypes.md) and [deck naming research](research/deck-naming.md). Directly checked pages verify Ceruledge and Dragapult Blaziken as published whole-deck labels. They do not establish that each named Pokémon is a separate archetype or that names have at most two components. Naming terms remain a user decision. Match-outcome rules are under research.

## Open decisions

- How deck names, archetypes and key Pokémon relate, including whether one or two components is a naming convention or a product limit.
- Who creates and maintains the naming choices, and how to handle custom or unknown choices.
- Game outcomes, time limits, incomplete matches and how to determine match results.
- Unknown opponent Decks, game-level details and how many unfinished matches may exist at once.
- Editing and deleting records, historical labels and desired summaries.

## Design constraint

The existing glossary uses a deck as an example of a physical Collection. The new match-tracking concept must not change owned quantities or treat a Proxy as an owned Card.

## Approval

Do not implement until the interview resolves the open decisions and the user confirms the design.
