# Pokémon TCG deck archetypes and match history

## Findings

- Pokémon’s tournament rules define deck construction and require submitted deck lists at Championship Series events; they do not define a separate “archetype” field or claim that an archetype names an exact list. The official rules and resources page is the source for current tournament documents: https://www.pokemon.com/us/play-pokemon/play-pokemon-tournaments/rules
- Limitless separates archetype/metagame pages from tournament decklists. Its archetype pages group tournament performance under labels, while list pages retain player, event, placement, and individual lists: https://limitlesstcg.com/decks and https://limitlesstcg.com/decks/lists
- Play Limitless makes the community judgment in grouping explicit: its deck statistics offer “Combine related deck variants into one archetype.” This shows archetype boundaries are editorial conventions, not a unique identifier derived from the exact list: https://play.limitlesstcg.com/decks
- Concrete dated example: Limitless’s 2023–24 Standard statistics list Charizard ex and Gardevoir ex as large archetypes; at the 2024 World Championships its results use the more specific label “Charizard/Pidgeot” and “Gardevoir.” These labels can name a key Pokémon and, where useful, a partner/engine: https://limitlesstcg.com/decks?format=all&region=all&time=2324&type=all ; https://www.limitlesstcg.com/tournaments/420/cards
- Archetype, a player-owned deck, and a particular deck list are distinct concepts. An archetype is a broad strategy/family; a user-owned deck is the player's continuing identity for a deck they maintain; an exact list is the card-by-card snapshot used at a time or event. The official tournament list requirement supports recording the exact list separately when available; Limitless's separate archetype and list views support this distinction.
- Lists change over time, and related variants can be grouped or split. Keep the archetype label and the list snapshot tied to a match/event date rather than assuming the player's current list was used in an older match. Limitless's variant-combining option and per-event tournament lists illustrate this distinction.
- Hybrid and rogue lists may not fit a stable published label. A community taxonomy can place related variants together, but the cited sources do not establish a universal rule for hybrid or rogue classification. Record a descriptive name or “unknown/other” rather than asserting a canonical archetype when evidence is weak.
- Naming commonly uses a prominent Pokémon, sometimes with a second Pokémon/engine to distinguish a variant (for example, Charizard/Pidgeot). The cited tournament and taxonomy pages show these naming patterns, but do not define a universal naming standard.
- Format, region, and time filters on Limitless archetype pages show that an archetype's prevalence and meaning are scoped to a format, region, and period. A match record should retain those scopes when relevant; identical names need not imply the same list across formats or eras: https://limitlesstcg.com/decks?format=standard&game=PTCG
- If a player does not know an opponent's exact list, an archetype is only an observed/estimated label. The official list requirement concerns tournament registration, not what an opponent can know during a match. The sources do not specify an opponent-identity convention; uncertainty should remain explicit rather than converting a guess into an exact list.

## Open decisions for the product discussion

- Whether match history links to an owned deck, an exact dated list, an observed opponent archetype, or more than one of these.
- How to represent an uncertain opponent label and whether the user can leave it unknown.
- Whether names need explicit format/region/time context, and how to handle hybrids or unpublished decks.

## Limits

- This research uses official Pokémon tournament resource links and first-party Limitless taxonomy/results pages. It does not establish a universal community taxonomy or inspect every publisher's naming practice.
- Limitless pages are live and can change; the dated 2023–24 and 2024 examples above provide historical anchors, but the current pages may display updated data.
- No claim is made that an archetype uniquely identifies a player's deck or exact list.
