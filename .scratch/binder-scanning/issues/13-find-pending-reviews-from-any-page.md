# 13: Find pending reviews from any Page

**What to build:** A user never forgets a pending scan on a Page they are not looking at. Directly under the jump form, visible on every Page, a notice lists the Pages that have a pending Page Review ("Pages 4, 7 have a pending scan"), and each Page number is a link that jumps to that Page, where ticket 12's banner offers Resume and Discard. Pressing "Scan this Page" on a Page that already has a pending review asks "Resume or start over?" instead of silently replacing it.

Spec: `.scratch/binder-scanning/spec.md` (Pending Page Review (frontend), stories 79 and 80). Frontend only. The notice sits in the binder view only; no marker goes on the Collections list.

**Blocked by:** 12

**Status:** ready-for-agent

- [ ] With no pending review in this binder, no notice is shown
- [ ] With pending reviews on some Pages of this binder, a notice under the jump form on every Page lists exactly those Pages, in order, and only for the signed-in user and this binder
- [ ] Each Page number in the notice jumps to that Page, and the notice updates when a review is confirmed or discarded
- [ ] "Scan this Page" on a Page with a pending review asks to Resume or start over; Resume opens it, and start over discards it only once the user chooses that, then goes to the photo step
- [ ] The frontend tests (generated client mocked, IndexedDB faked) cover the notice's contents and links, its disappearance after Confirm and Discard, and both answers of the resume-or-start-over prompt
- [ ] Build, lint, format and unit tests pass
