# Decks and match records

Status: awaiting final design approval; do not implement yet.

## Scope

- MVP supports private Pokémon TCG Decks and match records for Indonesian Standard play, separate from Collections and owned quantities.
- Card lists, Proxy support, structured tournament management, sharing, performance statistics and matchup summaries are outside MVP.
- Future Deck lists must allow Proxies without treating them as catalog Cards or owned Cards. Structured tournament management is also a future goal; neither goal requires new infrastructure in this release.

## Deck identity and naming

- A personal Deck has one or more Key Pokémon selected from published catalog Pokémon Cards. There is no maximum of two. Selecting a Card does not record ownership or establish an exact Deck list.
- Group Key Pokémon by full printed name across art and Expansion Sets. Keep suffixes, forms and owner names distinct: Dragapult and Dragapult ex are different Key Pokémon. This groups names, not species or identical attack text; do not use the narrower same-name-and-art Reprint rule.
- Selected Key Pokémon names produce the default Deck name. An optional name override replaces the prominent name, but the Key Pokémon must still appear below it in smaller text. No separate archetype field is needed.
- Several personal Decks may use the same Key Pokémon. Distinct name overrides help identify builds but are not required.
- Archive and restore Decks with matches. Archived Decks cannot be selected for new matches; their existing matches remain viewable, editable and resumable. Permanently delete only Decks with no matches.

## Match entry

- Select the user's personal Deck and BO1 or BO3. Record during or after play; several unfinished matches can exist and each can be resumed.
- Record the played date, defaulting to today but editable, plus optional event name and match notes. Structured events and rounds remain outside MVP.
- Record opponent Deck details using the same Key Pokémon and optional name-override rules. Allow Unknown Key Pokémon and later corrections. These details belong only to that match, not to a reusable opponent Deck library.
- An optional opponent person's name is separate from the opponent Deck's name. No opponent Cardstack account or confirmation is required.
- Record every started Match Game in order, with Win, Loss, Draw or Unfinished from the user's view. Who went first (user, opponent or unknown) and game notes are optional. A game never started needs no invented record.

## Results and completion

- Finish Match is explicit; no automatic timer or automatic completion. Until then, the match remains In Progress.
- At Finish Match, compare recorded game wins to suggest Win, Loss or Draw. Draw and Unfinished games add no win. Thus 1–0 suggests Win and 1–1 suggests Draw, including when a final game is unfinished. Finishing early does not require inventing missing games.
- Allow a match-result override with a required reason for exceptions. Finishing with no played games requires an explicit result override and reason, such as a concession or agreed draw. Structured tournament byes remain outside MVP.
- Do not add games after either player reaches one win in BO1 or two wins in BO3. Drawn games before that target can make the game count exceed the usual one or three. Corrections must leave a valid ordered sequence; do not silently remove later games.
- Changes to game results or BO1/BO3 must recompute the suggested result. If a match-result override exists, require the user to reconfirm or clear it before saving a result-affecting change.

## History and corrections

- Show history newest first by played date, with filters for the user's Deck and In Progress or Finished status. No other history filters or statistics are required for MVP.
- Preserve the Deck details recorded in each match. Renaming a Deck or changing its Key Pokémon affects new matches, not old ones. Explicit match edits may correct the saved Deck details.
- Allow edits to completed matches, including Deck choices, individual games and match details. Keep a corrected match's computed result and any confirmed override consistent.
- Permanently delete a mistaken match and its games only after confirmation. Leave its Deck and other matches unchanged.
- Decks, matches, opponent names and notes remain private to their owner. This feature changes no Collection, Inventory Entry or Master Inventory.

## Research and limits

See [archetype conventions](research/archetypes.md), [verified deck naming](research/deck-naming.md), [catalog grouping facts](research/catalog-pokemon-grouping.md) and [regional match rules](research/match-results.md). The user chose Key Pokémon and full-name grouping as product language and policy; the sources do not establish a universal two-component archetype system.

This app records personal results; it does not judge tournament rules or assert that a selected Deck is currently legal. Event-specific timeout rules remain unverified. Indonesian Floor Rules Asia 1.2 section 2-3 restricts phone use during play unless staff expressly permit it, so live entry must not imply permission to use the app during sanctioned play. Future casual Proxy support must not imply official tournament legality.

The catalog code review confirmed Card name/category/image fields and filter DTOs, not every stored name or a complete species mapping. Reuse the existing published-catalog access rules. Exact selector layout, representative artwork and storage design belong to implementation; they must not change the agreed name-based identity or historical records.

## Acceptance scenarios

1. Different-art Cards with the same full name describe one Key Pokémon; names differing by ex or another printed modifier remain distinct. A name override leaves the Key Pokémon visible below it.
2. Two personal Decks can share Key Pokémon. Editing or archiving one leaves its past match details intact; its unfinished matches remain resumable.
3. BO3 can record Win, Draw, Loss and an Unfinished final game. Finish suggests a Draw from the tied win count; a different actual result requires an override and reason. No new game follows a player's second win.
4. An unplayed concession can finish with an override and reason, without fictitious games. Changing a result beneath an existing override requires reconfirming or clearing it.
5. An owner can resume, correct, filter or delete their matches without changing card ownership. Another user cannot access those records, including opponent names and notes.

## Approval

All interview decisions are settled. Await the user's confirmation of shared understanding before marking this spec approved. Approval authorizes the design only, not an implementation run.
