# Change monitor schedule

I want to simplify the way that monitor schedule are set up.

Currently, they are cron expressions which determine how often a monitor's checks should run.

There are a couple of problems with this approach:

1. Since it's always configured to run on the 0th minute, this creates a spike in checks that
   try to run all at the same time.
2. The UI for configuring the cron expression is too complicated.

I'd like to simplify by instead letting users configure a check frequency which takes effect
based on the time that they've activated their monitor.

## Storage

This should be stored as a `check_frequency_minutes` column on the `monitors` table and should replace
the existing `check_schedule` column.

Its default should be every 24 hours (daily).

It's okay to wipe out the existing data in the `check_schedule` column and drop it.

## UI

We should remove the current `ScheduleSelector` component from the Dev palette and replace it with
a simple select for now, that has the following options:

- every hour
- every 8 hours
- every day
- every 2 days
- every week

## HTTP layer

The existing handlers for updating the monitor's schedule should be renamed so their names reflect
the new name of the property being stored. Their implementations can match what's already there, but
should use the new property.

## Service layer

Update SQL queries to remove the unused ones that modified `check_schedule` and replace with ones
that modify `check_frequency_minutes`.
