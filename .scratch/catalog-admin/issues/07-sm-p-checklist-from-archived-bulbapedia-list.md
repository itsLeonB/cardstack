# 07: SM-P checklist from the archived Bulbapedia list

**What to build:** A file the owner can enter `SM-P` cards from: collector number, English name and category for all 186 cards, taken from the archived copy of Bulbapedia's ITCG list as facts only. Research found this is the only complete source (no live access: Bulbapedia is behind a bot challenge); the Wayback snapshot dated 2025-07-20 is fetchable as raw HTML and had 186 rows, 001 to 186, with no gaps.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [ ] A checklist file lists, for each row, the collector number (as `NNN/SM-P`), the English name and the Category, derived from the source's type code (blank means Pokémon; `I`, `Su`, `St` mean Trainer with Tag Item, Supporter, Stadium; `E` means Energi with an empty Tag)
- [ ] The Promotion text and per-card prose from the source are not copied
- [ ] Variant rows that share one source row (for example Indonesian Regional Stamp and Gym Stamp prints of one card) are flagged so the owner can decide how to enter them
- [ ] The file's header carries an attribution line naming Bulbapedia and its CC BY-NC-SA 2.5 licence, and the snapshot URL and date
- [ ] The header notes that English names are used until corrected from the physical card, and that rarity is "Tanpa Tanda" for all cards
- [ ] The source is fetched politely (about one request per second, no bulk scrape of other pages)
