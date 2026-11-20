# An open Trade is discarded by deleting it

A Trade nobody has joined has no history worth keeping, so "Discard" hard-deletes the Trade and its Trade Items while it is `open`, while "Cancel" soft-cancels a joined Trade and keeps it in the list. This replaces the single soft-cancel path in issue 07, where cancelling from `open` left a `cancelled` row with a dead slug.

## Considered Options

- Keep one soft-cancel path for every cancellable state. Rejected: an abandoned `open` Trade has no counterparty and no agreed deal, so a retained row is noise, and one button had to mean two different things.
- Hard-delete in every cancellable state. Rejected: a joined Trade carries an agreed deal and a history that must survive as a cancelled row.

## Consequences

- The backend needs a real delete for an `open` Trade, including its Trade Items; cancelling from `negotiating` and `locked` stays a status update.
- The Trades list button label depends on state: "Discard" while `open`, "Cancel" after a join.
- A discarded Trade's link dies with the row; a cancelled Trade keeps a dead slug so its old link never resolves again.
- Issue 07's rule that cancelled Trades stay as history now holds only after a join.
