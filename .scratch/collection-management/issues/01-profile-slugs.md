# 01: Profile slugs

**What to build:** every profile, new and existing, has a permanent Slug: the owner's name lower-cased and hyphenated, then a hyphen and the last 5 hex characters of the profile's id (GLOSSARY.md: Slug; ADR-0025). Nothing is visible to users yet; later tickets build the shared links on it. This is the prefactor for sharing.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] A migration adds a unique Slug to profiles and backfills it for every existing profile before the column becomes required.
- [ ] A new profile gets its Slug when it is created, and it never changes afterwards, including when the name is later edited.
- [ ] A name that slugifies to nothing (emoji or non-Latin only) falls back to the stem `user`.
- [ ] Two profiles with the same name get different Slugs, and a unique index backs the tail.
- [ ] HTTP-boundary tests against a real Postgres cover profile creation and a backfilled profile.
