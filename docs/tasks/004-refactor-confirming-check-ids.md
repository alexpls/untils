When a check completes, it will either create a new result or confirm an existing one.

This is tracked by the `monitor_results.confirming_check_ids` array, and the
`monitor_results.latest_confirmation_at` timestamp.

As we have accumulated more and more checks for each result, this has started to get
unwieldly, with the array containing dozens of entries, and with no cap to ever limit
it we should refactor this before it grows to hundreads - since querying this array
gets more and more expensive as our number of checks grow.

To support this, let's embark on a refactor:

## Database changes

We should:

- Create `monitor_checks.result_id` column (nullable) which is set to the corresponding
  result of a check.
- Migrate data from the old column to the new one.
- Remove the redundant columns on `monitor_results`.

Typically when we select a monitor's result we also want to know when it was last confirmed
at, and now we won't have that denormalized on the table anymore. We could join in all the
current queries that look for this column, but SQLc would then create a different struct
for each of those, which would be unnecessary duplication. (correct me if I'm wrong, but
I'm assuming that SQLc cannot have different queries share the same struct).

So we should then create a view like `monitor_results_with_latest_check` that joins `monitor_results`
with `monitor_checks` and does what we're after.

The goal is for the current `monitor_results` queries that also look at the latest check to use
this instead, and for SQLc to only generate one struct for these queries instead of many.

## Service layer changes

The changes from the database should flow through to the service layer, and the various parts
that currently update `monitor_results.confirming_check_ids` will need to update the `monitor_check`
instead.

## UI changes

You'll need to change UI internals (like the structs based on our new queries), but I expect there to
be no user facing changes made. If all goes well, things should look the same.
