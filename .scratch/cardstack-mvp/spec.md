Status: ready-for-agent

# Cardstack MVP

## Problem Statement

TCG collectors — starting with Indonesian Pokémon TCG players collecting Scarlet & Violet (SV) and Evolusi Mega (MA) — currently track which cards they own across multiple binders, boxes, and decks manually (spreadsheets, memory, or nothing at all). This makes it hard to answer simple questions: how many of this card do I own in total, and which of my binders/boxes/decks is it actually in right now?

## Solution

A web app where a user manages any number of named Collections (binders/boxes/decks), records how many of each Card Variant they hold in each Collection, sees a computed Master Inventory (total owned across every Collection), and can search the card catalog to find a card and see which of their own Collections it appears in.

## User Stories

1. As a new user, I want to register an account with an email and password, so that I have a private place to track my cards.
2. As a returning user, I want to log in with my email and password, so that I can access my Collections and Master Inventory.
3. As a user, I want my session to persist across visits without re-entering my password every time, so that the app is convenient to use daily.
4. As a user, I want to log out, so that I can end my session on a shared device.
5. As a user, I want to create a new Collection with a title, so that I can start tracking a new binder/box/deck.
6. As a user, I want to optionally add a description to a Collection, so that I can note what it's for (e.g. "trade binder", "SV starter deck").
7. As a user, I want to optionally set a maximum card-count limit on a Collection, so that it reflects a binder or box's real physical capacity.
8. As a user, I want an attempt to add cards past a Collection's capacity limit to be rejected, so that my tracked counts can't silently exceed what the container can physically hold.
9. As a user, I want to rename or edit a Collection's title/description/limit, so that I can correct or update it later.
10. As a user, I want to delete a Collection, so that I can remove one I no longer keep.
11. As a user, I want to be warned and asked to confirm before a Collection (or any data inside it) is deleted, so that I don't lose data by accident with no undo available.
12. As a user, I want to see a list of all my Collections, so that I can pick one to view or edit.
13. As a user, I want to browse the full contents of one Collection (which Card Variants, and how many of each), so that I know what's physically in that binder/box/deck.
14. As a user, I want to add a Card Variant to a Collection with a quantity, so that I can record cards I've placed into it.
15. As a user, I want to update the quantity of a Card Variant already in a Collection, so that I can reflect adding or removing physical copies.
16. As a user, I want to remove a Card Variant entirely from a Collection, so that my records match reality when I take a card out.
17. As a user, I want to search the card catalog by name, so that I can find a specific card regardless of which of the four locale names (Indonesian and others carried by the source data) I remember it by.
18. As a user, I want to filter search results by Expansion Set / card number, so that I can narrow down to the exact print I mean.
19. As a user, I want to filter search results by rarity, so that I can browse e.g. only the secret rares in a set.
20. As a user, I want search results to distinguish between print variants (normal, reverse holo, holo, first edition) of the same card number, so that I can find the specific physical variant I own or want.
21. As a user, I want to see, from a card's page, which of my own Collections contain it and in what quantity, so that I know where a specific card physically is without checking every binder.
22. As a user, I want to see my Master Inventory — the total quantity I own of every Card Variant across all my Collections combined — so that I know my full holdings without adding things up myself.
23. As a user, I want the Master Inventory to update immediately when I change any Collection's contents, so that it's always accurate without a manual refresh/sync step.
24. As a user, I want to browse Expansion Sets (SV, MA) and see all Cards within one, so that I can explore a set even for cards I don't own.
25. As a user, I want cards I don't own at all to still be visible/searchable in the catalog (distinct from my personal holdings), so that browsing and collecting are separate concerns.
26. As a maintainer, I want to seed the card catalog from TCGDex data for Indonesian SV sets via a CLI command, so that the catalog exists before users can record holdings against it.
27. As a maintainer, I want to seed the card catalog from a headless-browser scrape of the official Pokémon Asia site for MA sets, so that MA cards (which TCGDex doesn't source) are also in the catalog.
28. As a maintainer, I want both seed paths to be manually triggered (not scheduled), so that I control exactly when new set data lands, given sets release only a few times a year.
29. As a developer, I want the domain model (Game → Expansion Set → Card → Card Variant) to not assume any card or set is shared across regions/languages, so that adding Riftbound or EN/JP Pokémon sets later doesn't require migrating existing data.
30. As a user, all of the above should only ever show or affect my own data, so that my Collections and holdings stay private with no cross-user visibility in MVP.

## Implementation Decisions

- **Domain shape**: `Game` → `Expansion Set` → `Card` → `Card Variant`, generalized across games from the start (ADR-0001). `Expansion Set` and `Card` identity is scoped per print edition/region (uniqueness: Expansion Set by `(game, set_code)`, Card by `(expansion_set, local_number)`) — never assumed shared across regions, per the corrected `CONTEXT.md` glossary. Game-specific attributes (Pokémon HP/types/attacks) attach per-Game rather than as fixed columns on `Card`.
- **Collection**: user-owned, has a title, optional description, optional maximum card-count limit counted as *summed quantity* (not distinct variant count) and hard-enforced at write time. No type-specific rules for binder/box/deck in MVP — a type label, if present, is display-only.
- **Inventory Entry**: the atomic `(collection, card_variant) → quantity` record. This is the only place quantity is written.
- **Master Inventory**: not a stored table — computed on read as `SUM(quantity) GROUP BY card_variant` across a user's Inventory Entries (ADR-0002).
- **Auth**: `github.com/itsLeonB/go-authkit` in stateful mode (session + refresh-token rotation), wired via custom Huma handlers calling `authkit.Kit`'s core methods directly rather than the library's Gin-only `authgin` adapter (ADR-0003), so auth endpoints appear in the same generated OpenAPI spec the frontend codegens against. MVP implements registration and login only — no email verification, no password reset, no OAuth, and Resend is not integrated at all (ADR-0004).
- **Persistence**: PostgreSQL via GORM + `github.com/itsLeonB/go-crud` (`CRUDRepository[T]`, `Specification[T]`), goose migrations. Deletes are hard deletes (`go-crud` has no soft-delete support) — the frontend must confirm with the user before firing any delete.
- **Ingestion**: two standalone CLI commands (not a runtime service), manually triggered by a developer: one pulling TCGDex's Asia-region SV data, one driving a headless browser against the Pokémon Asia MA card-search pages (the listing is JS-rendered — no usable JSON/XHR endpoint was found, so a real page load is required rather than a raw HTTP fetch).
- **API contract**: Huma generates the OpenAPI spec (including auth routes); the frontend runs `orval` against it to generate zod schemas and TanStack Query hooks in one step.
- **Search**: matches card name in any locale the source data carries, with Expansion Set/card-number and rarity as additional filters; results are per Card Variant, not per Card, so print finish is distinguishable.
- **Observability**: OpenTelemetry tracing is scaffolded in the backend (instrumentation points present) but disabled for MVP — no otel-collector deployed yet, enabling it later should not require code changes.
- **Infra**: Neon Postgres as the single provider for both production and PR-preview branches (previews are Neon branches of the prod project); Railway (backend) + Vercel (frontend), each with a per-PR preview deploy wired to its Neon branch via an existing GitHub Action; GitHub Actions for CI.

## Testing Decisions

Unit tests cover a single component and its edge cases; feature tests cover one functionality end-to-end, or a cohesive set of functionalities, testing their interaction (ADR-0005 and follow-up discussion).

- **Backend repository unit tests**: run against a real local Postgres instance (not mocked) to verify actual `go-crud` spec/scope/query behavior. CI provisions a temporary Postgres for the same tests.
- **Backend service unit tests**: mock the repository *interfaces* using `mockery`-generated mocks, so business logic (capacity-limit enforcement, Master Inventory aggregation, etc.) is tested without a DB dependency.
- **Backend feature tests**: exercise the Huma HTTP handler boundary (request in, response out) against the full real stack, including the same real Postgres instance — no mocking at this layer.
- **Frontend feature tests**: render the route/page tree with the orval-generated API client mocked via `vi.mock` (no MSW — an added dependency isn't justified when mocking the generated client module directly already covers it).
- **Frontend unit tests**: individual components/hooks in isolation, using the already-scaffolded Vitest + Testing Library setup.
- No prior art exists yet in either codebase (both are bare scaffolds) — the first test written in each layer establishes the pattern subsequent tests should follow.

## Out of Scope

- Wishlist / "wanted but not owned" tracking.
- Cross-user visibility, public or shared Collections.
- Email verification, password reset, and OAuth login (all deferred post-MVP per ADR-0004).
- Riftbound, English/International, and Japanese Pokémon Expansion Sets (roadmap items, not this spec).
- Binder page/slot layout tracking.
- Deck-specific rules (format legality, exact deck-size enforcement) — a deck is just a Collection with an optional capacity limit, no format validation.
- Soft delete / undo for any entity.
- A staging environment (MVP is prod + per-PR preview only).
- An actually-running otel-collector or trace backend (tracing is scaffolded, not wired to a destination).
- Physical condition grading (NM/LP/MP/etc.) or per-copy language tracking within a Card Variant.

## Further Notes

- Domain vocabulary is maintained in `CONTEXT.md` at the repo root; architectural decisions referenced above are recorded as ADR-0001 through ADR-0005 in `docs/adr/`.
- The most consequential domain correction made during design: an Expansion Set is never assumed to be shared across regions/languages, even though the TCGDex source data for Indonesian SV bundles multiple locale names onto one shared file per print edition. That bundling is a detail of that one source, not a general rule — a future Japan or English Expansion Set gets entirely its own Cards.
