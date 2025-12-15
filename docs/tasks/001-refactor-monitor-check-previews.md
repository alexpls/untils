# Refactor monitor_check_previews

Currently as defined in the migration at internal/db/migrations/20251130003952_create_monitors.up.sql we have monitor_check_previews which represent the results of a check prior to the monitor being active (e.g. when it's still in draft and being worked on by the user).

I'd like to roll this behvaiour into the existing monitor_checks table to avoid duplication and to allow for richer experiences with setting up the draft monitor that will require more columns representing the check results. Keeping these as two separate tables would this harder to do and add more duplication.

However a key difference between how monitor_check_previews and monitor_checks currently work is that only one preview is selected during the monitor's draft phase and is then turned into the first check.

With this refactor, for now I'd like to focus on keeping the same behaviour but removing the monitor_check_previews table.

So:

1. Delete the `monitor_check_previews` table from the existing migration file
2. Run `mise run db-reset` to rerun the migrations and regenerate SQLc schema
3. Refactor the codebase to use the new concept

The part you'll need to pay careful attention to is activating the monitor from the check result. After the refactor I expect you to delete all check results for the monitor besides the one that was picked.
