# 02: Record, resume and finish matches

**What to build:** Let a user record a private match during or after play, resume it later and explicitly finish it with its actual result. This slice must work from selecting a personal Deck through persisted Match Games and a finished history entry.

**Blocked by:** 01: Create and edit personal Decks.

**Status:** ready-for-agent

## Entry and resumption

- [ ] Start a BO1 or BO3 match using the user's personal Deck. Record an editable played date defaulting to today, optional event name and optional match notes. Allow several In Progress matches and resume any of them after reload.
- [ ] Record the opponent's Key Pokémon with the same grouping and naming rules as personal Decks, including an optional Deck name override with smaller Key Pokémon text. Allow Unknown Key Pokémon and later entry. Keep these details within the match, not a reusable opponent Deck library.
- [ ] Keep an optional opponent person's name separate from their Deck name, without requiring their Cardstack account or approval.
- [ ] Save the Deck details used in the match. Later changes to the personal Deck do not rewrite those details; matching two personal Decks by their Key Pokémon must not merge their identities.
- [ ] Display saved matches newest first by played date with In Progress or Finished status and enough details to open a match. Filtering and deletion belong to later tickets.

## Games and finishing

- [ ] Persist every started Match Game in order, with Win, Loss, Draw or Unfinished from the user's view, optional who-went-first (user, opponent or unknown), and optional game notes. Do not invent records for games never started.
- [ ] Permit game entry and corrections while the match is In Progress. Prevent another game after either player reaches one win in BO1 or two wins in BO3. Drawn games before the target may extend the count beyond one or three; reject invalid corrected sequences without silently dropping later games.
- [ ] Keep the match In Progress until the user selects Finish Match. Do not add a timer or automatic completion. Compare recorded game wins to suggest Win, Loss or Draw; Draw and Unfinished games add no win, so 1–0 suggests Win and 1–1 suggests Draw.
- [ ] Allow a different match result only with an explicit override and required nonblank reason. A match with no played games can finish only with an explicit override and reason, such as a concession or agreed draw, without fabricated games.
- [ ] Persist the finished result, game records and any override reason and show them after reload. Correction of an already Finished match belongs to ticket 03, not this slice.

## Safety and proof

- [ ] Only the owner can access or change matches, games, opponent names or notes, including direct requests. Match recording changes no owned quantities.
- [ ] Automated checks cover persisted entry/resumption, saved Deck details surviving a Deck edit, valid game sequences, score suggestions, required override reasons and owner isolation. Browser checks demonstrate resume and finish, including an exceptional ending.
- [ ] Treat BO1/BO3 and the agreed score calculation as personal tracking policy, not a tournament legality or timeout-rule engine. Live entry must not imply that a sanctioned event permits phone use. Exclude structured events, byes, sharing and statistics.
