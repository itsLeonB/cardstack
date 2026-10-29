# 03: Shared binder view

**What to build:** a Shared Collection that has a Layout shows its Pages and Slots read-only, as the owner sees them, and the Unidentified count, so a viewer sees how the binder is laid out.

**Blocked by:** 02 (Share a Collection and open its link).

**Status:** done

- [x] The shared read for a binder Collection carries its Layout, its Pages with their Slots (empty, holding a Card, or Unidentified) and the Unidentified count, and still never the email or any editing capability.
- [x] The shared page shows the binder as Pages of Slots, read-only, in place of or beside the list, following the owner's binder view.
- [x] Unidentified Slots show as a count and not as a Card. Slots holding Cards of unpublished Expansion Sets are not revealed.
- [x] HTTP-boundary tests cover a Shared binder as Guest and owner, and screen tests cover the read-only binder view.

**Note:** the branch adds service and repository tests (`binder_page_service_test.go`, `shared_collection_service_test.go`, `binder_slot_repository_test.go`) on top of the HTTP-boundary tests, against the spec's "no repository-level or service-level tests" line. This is deliberate: `general.md` has unit tests own branches, and the repository cases pin the unpublished-set SQL filter directly.
