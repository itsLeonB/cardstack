# 40: Adapter SQL is executed by a test against the pgvector database

Type: task
Status: done

## What to build

The filtered nearest-cards scan runs in a test against the stack's pgvector `db`, settings and all, so an invalid `SET LOCAL hnsw.iterative_scan` value (or an unhonoured `ef_search`) fails the test locally instead of reaching CI or production. The first such test is the scan path that shipped broken in 80ed76e; the rule it establishes — every adapter SQL path gets a test that executes it against the real database — is recorded in the backend conventions so the next one is written without a review comment asking for it.

## Blocked by

None (can start immediately).

## Context

PR #79 changed the scan to set `hnsw.iterative_scan = 'strict_order'` through a raw `SET LOCAL`. The initial value was invalid; nothing in the backend suite executes that statement, so `scripts/verification/backend.sh test` was green. The only HNSW assertion checks that the index exists.

The stack's `db` is `pgvector/pgvector:pg18`, and the migration tests already bring a migrated database up through a shared `TestMain` and skip when it is unreachable — that harness is the place to start.

## Acceptance criteria

- [x] A test executes the filtered nearest-cards scan end to end against the database, with the filtered and unfiltered cases distinguished.
- [x] Temporarily replacing `strict_order` with an invalid value makes that test fail; reverting makes it pass. Record both outcomes in the PR body.
- [x] The test runs under `scripts/verification/backend.sh test`, needs no database beyond the stack's `db`, and skips the same way the existing migration tests skip without one.
- [x] `docs/agents/conventions/backend.md` states the rule and names this test as its example.

## Comments

No test in `internal/domain/repository` or the migration package skips when the database is unreachable: `testDB` and the migration `privateDB` both fail instead, and `scripts/verification/backend.sh test` brings `db` up itself. The new test follows that existing harness and fails the same way rather than adding a skip nobody else has.
