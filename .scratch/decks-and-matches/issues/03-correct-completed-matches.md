# 03: Correct completed matches

**What to build:** Let the owner correct a Finished match without losing its history or leaving its result inconsistent with its games.

**Blocked by:** 02: Record, resume and finish matches.

**Status:** ready-for-agent

- [ ] Edit and persist a Finished match's played date, event name, match notes, opponent person's name, opponent Deck details and the user's Deck choice. Correct saved Deck details only through an explicit match edit; unrelated personal Deck edits must not rewrite history.
- [ ] Correct ordered game records, game results, who went first, game notes and BO1/BO3. Keep the agreed win-target sequence valid and reject invalid sequences without silently deleting later games.
- [ ] Recompute the suggested match result when games or BO1/BO3 change. If a result-affecting change encounters an existing override, require the owner to reconfirm or clear it before saving; retain a required reason for any confirmed override.
- [ ] Keep the corrected match Finished and show its saved details and result after reload. Allow existing matches to retain an archived Deck once archiving is available. Reject other users' changes without altering the record or card ownership.
- [ ] Automated checks cover a corrected normal result, override reconfirmation and clearing, rejection of invalid sequences without partial changes, saved Deck details and owner isolation. A browser journey corrects a Finished match and verifies the saved result.
