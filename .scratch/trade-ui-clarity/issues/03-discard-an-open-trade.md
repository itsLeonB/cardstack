# 03: Discard an open Trade

**What to build:** An owner who made a Trade by mistake and has no counterparty yet can remove it completely. Discard deletes the Trade and its Trade Items after confirmation, so the link stops working and no cancelled row is left in history. Once someone has joined, the familiar Cancel still ends the Trade and keeps it in history.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] The owner of an `open` Trade sees a Discard action; a participant in a `negotiating` or `locked` Trade keeps Cancel; neither action is offered for an `executed` Trade.
- [ ] Discard asks for confirmation in the house destructive dialog, names the destructive action, uses danger emphasis, and puts focus on the safe action by default.
- [ ] Discard and Cancel are visually separated from ordinary Trade actions such as Edit and Accept.
- [ ] Confirming Discard deletes the Trade and its Trade Items; the Slug answers not found at once and no cancelled row appears in the Trades list.
- [ ] Cancelling from `negotiating` and `locked` keeps the existing cancelled state, dead Slug and history row; cancelling from `executed` is still refused with the same stable refusal.
- [ ] The Trades list refreshes after a Discard, and a failed delete is reported plainly with the Trade still present.
- [ ] The backend gains a delete operation for an `open` Trade that removes the Trade and its Trade Items, while the existing status-update path for Cancel is unchanged.
- [ ] A backend route case proves an `open` Trade is deleted and its old Slug answers not found; the existing cancel-from-each-state cases stay as the proof for joined Trades.
- [ ] Guest and non-participant access rules, private indexing and the no-notification behaviour are unchanged.
- [ ] Focused checks and the full backend, frontend and authenticated E2E results are recorded on final source with actual exit status and any retries.
