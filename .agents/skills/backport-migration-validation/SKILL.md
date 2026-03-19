---
name: backport-migration-validation
description: Backport the staging database into local untils_dev and validate that migration changes apply cleanly, including data migration integrity checks (field casing, schema coverage, and linkage sanity). Use when changing or reviewing DB migrations in untils.
---

# Backport Migration Validation

Use this workflow to validate migration changes against staging-like data.

## Workflow

1. Ensure migration SQL changes are in place before validation.
2. Run `./scripts/backport-staging-db.sh` to refresh `untils_dev` from staging.
3. Check starting version:
   `/bin/zsh -lc "migrate -database ${PG_URL/postgresql/pgx5} -path internal/db/migrations version"`
4. Run migrations:
   `mise run db:migrate:up`
5. Check ending version with the same `migrate ... version` command.
6. Run data integrity checks with `psql postgresql://root:root@localhost:54324/untils_dev -Atc`:
   - Row counts for core tables (`monitors`, `monitor_results`, `monitor_checks`, `monitor_result_checks`, `monitor_schemas`).
   - Coverage checks:
     - `monitors` without `monitor_schemas`
     - `monitor_results` without `monitor_result_checks`
   - FK/orphan checks:
     - `monitor_result_checks` rows missing parent result/check
     - `monitor_results.last_confirmed_check_id` pointing to missing checks
   - Migration-shape checks:
     - each result has a `Result` text field
     - no empty `headline`
     - no empty `data.fields`
   - Field-name casing checks for migrated date fields in both `monitor_results.data` and `monitor_schemas.data`:
     - count names starting uppercase
     - count names starting lowercase
   - Subtitle consistency check:
     - compare subtitle placeholder (`{{...}}`) with date field name in `data`.

## Expected Outcome

- `mise run db:migrate:up` completes without migration errors.
- Final migration version matches latest migration.
- Integrity checks show zero missing links/orphans/mismatches.
- Date field names migrated from legacy values start with uppercase first letter.

## Notes

- Keep SQL lowercase in ad-hoc queries.
- If local DB connectivity is blocked by sandbox policy, rerun the exact command with escalation.
