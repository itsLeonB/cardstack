# Decks and match records

Status: design interview in progress; not approved for implementation.

## Confirmed scope

- First release: Pokémon TCG, Indonesian Standard play.
- Deck management starts with named personal Decks, without card lists or changes to owned quantities. The user confirmed separate named Decks in round 2.
- Each Deck has structured Key Pokémon choices that produce its default descriptive name, such as Dragapult Blaziken or Ceruledge. The user may override the displayed name; when they do, still show the Key Pokémon below it in smaller text. No separate archetype field is agreed.
- Record every individual game in a match, not only the overall match result.
- Support best-of-one and best-of-three matches, with the match result determined from game results.
- Record the opponent's Deck (described by the user as one or two main archetypes), plus an optional person's name. No opponent Cardstack account is required.
- Allow match entry during or after play; unfinished matches can be saved and resumed.
- Each match has a date, optional event name and optional notes. Structured tournament management is a future goal, outside this release.
- Decks and match records are private to their owner in this release.
- Each Match Game has a result, optional who-went-first (user, opponent or unknown), and optional notes.
- MVP shows match history only, without performance summaries or matchup statistics.
- Completed matches can be edited, including Deck choices and individual game results; update the resulting match outcome when affected.
- Future deck lists must allow Proxies, which are not catalog Cards. Proxy support is outside this first release.

## Research

See [archetype research](research/archetypes.md) and [deck naming research](research/deck-naming.md). Directly checked pages verify Ceruledge and Dragapult Blaziken as published whole-deck labels. They do not establish that each named Pokémon is a separate archetype or that names have at most two components. The user chose Key Pokémon as the component term. See [match-result research](research/match-results.md) for directly checked regional general rules and the event-specific gaps.

## Open decisions

- Who creates and maintains Key Pokémon choices, how many each Deck can have, and how to handle unknown opponent Decks.
- Game outcomes, time limits, exceptions to computed match results and how to finish a match.
- How many unfinished matches may exist at once, history order and filters.
- Deleting matches, archiving Decks, duplicate Deck names and preserving Deck details in past matches.

## Design constraint

The existing glossary uses a deck as an example of a physical Collection. The new match-tracking concept must not change owned quantities or treat a Proxy as an owned Card.

## Approval

Do not implement until the interview resolves the open decisions and the user confirms the design.
