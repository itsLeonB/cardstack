# Shared pages are link-only and identified by stored slugs

A Collection or a Wishlist can be Shared: anyone with its link, signed in or not, can view it read-only. Sharing is off by default and is the owner's explicit choice. The link has two path segments, `/s/<profile slug>/<collection slug>` or `/s/<profile slug>/wishlist`, and each segment is looked up by a slug stored on its row rather than derived from the name or parsed out of a combined string. A profile always has a Slug, made once from the name and never changed. A Collection has a Slug only while Shared: it is made when sharing is turned on, stays fixed while Shared so renaming the title does not break the link, and is cleared to NULL when sharing is turned off, so a NULL Slug is what makes a Collection private. A Wishlist has a plain `shared` boolean, since there is one per user at a fixed path.

Each Slug is the lower-cased, hyphenated name or title followed by the last 5 hex characters of the row's own id. The first 5 would not do: ids are `uuidv7()`, so their leading characters are a millisecond timestamp that stays the same for about three days, and rows made in the same window would all share it. The last 5 are random, and a unique index backs the tail. A name or title that slugifies to nothing falls back to `user` or `collection`.

Shared pages carry `noindex` and appear in no sitemap, show the owner's full name and never their email, and answer a private, unshared or nonexistent link with one identical not-found so nothing leaks. They are served from a guest-allowed route group with its own rate limit.

## Considered Options

- One combined hyphenated segment `<name>-<id5>-<title>-<id5>`, resolved by the two id tokens. Rejected: both slugs contain hyphens, so there is no unambiguous split point, and the server would have to try every 5-hex segment as the profile token.
- First 5 characters of the id as the disambiguating token. Rejected: uuidv7 makes them nearly constant across rows created in the same window.
- A separate `is_shared` boolean on Collection next to a slug. Rejected: two columns that can disagree. A NULL Slug meaning private is the one source of truth.
- Regenerating a Collection slug whenever the title changes. Rejected: a link that someone has already been given would break on a rename.

## Consequences

Links are hard to take back once given out in the world: unsharing kills the URL, but sharing again produces a new one, so an old link never comes back. Because a profile Slug never changes, a user's full name as it was at profile creation stays in every link they have shared, even if the name is later edited; fixing that needs a way to regenerate profile slugs that is out of scope here. The migration must backfill a Slug onto every existing profile.
