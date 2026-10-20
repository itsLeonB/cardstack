# 10: Ingest Cards whose printed number has no digits

**What to build:** The TCG Collector ingester ingests a Card whose printed number has no digits before the `/`, instead of failing it. Today `cardNumber` keeps only digits, so the three Mew in MA6 (`R/RGB`, `G/RGB`, `B/RGB`, the ultra rare "RGB" cards) fail with `number "R/RGB" has no digits` and the run exits 1. These are the only cards of this kind seen so far. The other printed-number kinds already work and do not change: `number/number` (normal), `higher/number` (secret rares such as AR, SR, SAR) and `number/set code` (promos, e.g. `001/SM-P`) all reduce to the integer before the `/`.

**Blocked by:** 06

**Status:** ready-for-agent

**Decided (grilling, 2026-10-20):**

- The rule fires when the text before `/` has no digits, not on a hardcoded `RGB` or Mew check, so a future card of this kind needs no code change. Digit-bearing numbers, including `No. 201`, keep today's integer path.
- The stored `local_id` is the whole printed number, trimmed and not zero-padded: `R/RGB`. The Asia ingester would store `R` for the same card (`splitCollectorNumber` cuts at `/`), so Sources do not agree on identity for this kind. Accepted: the catalog holds none of these cards today, and fixing a duplicate later is an Admin edit or a manual database change.
- A text ID matches an existing Card by exact, case-sensitive string against the set's local IDs. The integer `held` map is unchanged.
- The same text ID twice in one card list fails the second one with a named message; it is not silently skipped.
- Rarity, category and the other card-page fields come from the card page as for any Card. Nothing changes there.

- [ ] Tests first, red before green: a card list with an `R/RGB` Card, run with apply, adds it with `local_id` `R/RGB` and its rarity from the card page; the same list in a dry run reports it under added and writes nothing; a rerun adds nothing and counts it held; a set that already holds `R/RGB` counts it held and fetches no card page for it; two `R/RGB` entries in one list add one and fail the other with a message naming the card and number
- [ ] `TestCardNumber_ReducesPrintedNumbersToIntegers` still passes unchanged for every existing case, and a new case shows `R/RGB` is not an integer number (the text-ID branch is chosen by the caller, not folded into `cardNumber`'s integer result)
- [ ] `setCards` no longer reports `has no digits` for a number such as `R/RGB`; an empty number still fails as before
- [ ] A numeric Card in the same set still pads to the set's numeric width, and `numberWidth` ignores text IDs as it does now
- [ ] ADR-0021's identity paragraph and the Sources paragraph in `GLOSSARY.md` say that a number with no digits before the `/` is stored whole, unpadded and matched exactly, and ADR-0021's Consequences records that Asia would store such a card under the text before the `/`, so a later Asia publish would add a second row
- [ ] Spec story 30/31 in `.scratch/tcgcollector-ingestion/spec.md` and ticket 06's number bullet point at this ticket for the non-numeric case
- [ ] Backend verification script passes

## Comments

Branch `fix/ingest-tcgcollector-textual-numbers` was created from `main` during the grilling and holds no commits. The live MA6 run against the real catalog is ticket 11.
