# Cancel stale monitor jobs

When a draft monitor subject changes, existing in-flight work can become stale.
Currently that stale work still runs and only gets rejected at completion time.

Similarly, when a monitor is deleted, any queued or running jobs for that monitor
should be canceled as soon as possible.

This wastes compute and tokens, as jobs that we know are invalid still run to
completion.

This task documents an approach to make stale monitor work cancelable.

## Goals

- Cancel stale jobs early instead of waiting for completion.
- Avoid retries for stale jobs (subject changed, monitor deleted).
- Keep data model and runtime behavior simple.
- Preserve existing subject mismatch validation as a safety net.

## Non-goals

- No UI redesign required.
- No broad queue-level pause/resume logic.
- No monitor check status expansion required for this pass.

## Current behavior summary

- Draft updates enqueue `validate_draft` jobs.
- Check scheduling enqueues `check` jobs, but `monitor_checks` does not store
  River job ids.
- Stale protection is mostly at write time (subject mismatch), after expensive
  work already ran.
- Monitor-not-found handling in check execution is inconsistent and can cause
  retry behavior where cancellation is expected.

## Recommended approach

### 1) Add monitor-scoped job cancellation (via river args)

Add a service helper, for example:

- `cancelMonitorJobsTx(ctx, tx, monitorID int64) error`

Within this helper:

- Cancel check jobs by joining `river_job` to `monitor_checks` through
  `river_job.args->>'monitor_check_id' = monitor_checks.id::text`, filtered by:
  - `river_job.kind = 'check'`
  - non-finalized River job states
  - `monitor_checks.monitor_id = $monitor_id`
- Cancel `validate_draft` jobs for that monitor id by querying `river_job`
  where:
  - `kind = 'validate_draft'`
  - `args->>'monitor_id' = $monitor_id::text`
  - state is not finalized

Use River APIs (`JobCancelTx`) for cancellation.

### 2) Trigger cancellation at state-changing points

Call monitor-scoped cancellation before these operations:

- Draft update/revalidate path.
- Monitor delete path.
- Monitor pause path (including currently running checks).

Then proceed with current logic:

- For draft updates: enqueue fresh `validate_draft`.
- For deletion: delete monitor and related rows.

### 3) Use delete (not `invalidated`) for stale queued checks

For this project preference, deleting stale checks is acceptable.

Recommended sequence:

- Cancel River job first.
- Delete stale check rows.

This keeps behavior simple and avoids introducing a new check status enum value.

Decision:

- Stale check history is not needed for now.

### 4) Ensure stale conditions are non-retryable

Worker behavior should cancel (not retry) on stale conditions:

- Monitor missing.
- Subject mismatch / resource changed during execution.

Map those cases to `river.JobCancel(...)` in workers so the job finalizes as
cancelled.

## SQL changes

Add SQLc queries for:

- Listing `check` River job ids by monitor id via join on
  `args->>'monitor_check_id'`.
- Listing `validate_draft` River job ids by monitor id via
  `args->>'monitor_id'`.

If deleting stale checks in bulk:

- Add query to delete scheduled/checking checks for a monitor after cancellation.

Use lowercase SQL.

## Concurrency and race notes

- A job may start between selection and cancellation. River cancellation should
  still mark it for cancellation and notify running worker context.
- Keep subject mismatch validation in place as last-line safety.
- Cancellation helper should be idempotent.
- For delete flow, cancel jobs before deleting monitor/check rows so the
  join-based check job lookup can still find relevant River jobs.

## Tests

Add or update tests for:

- Draft subject changed while previous validation/check is queued -> old job
  canceled and stale check deleted.
- Monitor deleted with queued/running jobs -> jobs canceled, no retries.
- Worker returns cancel for missing monitor and stale subject mismatch.

## Rollout sequence

1. SQLc queries for job lookup by River args.
2. Service helper for cancellation.
3. Wire cancellation into draft update + delete + pause.
4. Worker stale error handling adjustments.
5. Tests.
