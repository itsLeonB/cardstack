# 03: Shared binder view

**What to build:** a Shared Collection that has a Layout shows its Pages and Slots read-only, as the owner sees them, and the Unidentified count, so a viewer sees how the binder is laid out.

**Blocked by:** 02 (Share a Collection and open its link).

**Status:** ready-for-agent

- [ ] The shared read for a binder Collection carries its Layout, its Pages with their Slots (empty, holding a Card, or Unidentified) and the Unidentified count, and still never the email or any editing capability.
- [ ] The shared page shows the binder as Pages of Slots, read-only, in place of or beside the list, following the owner's binder view.
- [ ] Unidentified Slots show as a count and not as a Card. Slots holding Cards of unpublished Expansion Sets are not revealed.
- [ ] HTTP-boundary tests cover a Shared binder as Guest and owner, and screen tests cover the read-only binder view.
