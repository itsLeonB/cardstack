# 03: Accept and lock

**What to build:** Each participant accepts the deal as it stands. When the second acceptance lands, the Trade locks and both sides freeze, so the set the deal is about cannot drift between agreeing and meeting. Until then the room shows who is still to accept.

**Blocked by:** 02.

**Status:** ready-for-agent

- [ ] One side's acceptance leaves the Trade in `negotiating`.
- [ ] The second acceptance moves the Trade to `locked`.
- [ ] While `locked`, adding or removing a Trade Item on either side is refused with a stable code.
- [ ] Accepting again from a side that already accepted changes nothing and does not error, so a double click is harmless.
- [ ] The read tells each participant whether their own side has accepted and, when it has not, that the Trade is waiting on them.
- [ ] Accepting is refused from `open` (no second participant), `executed` and `cancelled`.
- [ ] Routes tests cover the two-step acceptance, the freeze that follows, the repeat acceptance, and the refusal in each other state. The Trade page shows the accept control and the waiting state.
