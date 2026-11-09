# 07: Trades list and cancel

**What to build:** With no notifications, the Trades list is how a participant finds a Trade again after closing the tab. It sits in the account area, shows each Trade's state and how long it has stood, and offers cancel: an abandoned or mis-shared Trade ends by being cancelled, and its link stops working for good.

**Blocked by:** 03.

**Status:** ready-for-agent

- [ ] The Trades list shows only Trades the caller is a participant in, each with its state and its age.
- [ ] Executed and cancelled Trades stay in the list as history.
- [ ] Cancelling is allowed from `open`, `negotiating` and `locked`, by either participant.
- [ ] Cancelling from `executed` is refused with a stable code.
- [ ] After cancelling, the Trade's slug answers not found, and a new Trade draws a new slug so the old link never comes back.
- [ ] Routes tests cover the list's scoping, cancel from each state, the refusal from `executed`, and the dead slug. The account area shows the list with each Trade's state.
