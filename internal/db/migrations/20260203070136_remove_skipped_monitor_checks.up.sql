-- Delete skipped records first
delete from monitor_checks
where status::text = 'skipped';

-- Convert column to text to drop the enum
alter table monitor_checks
alter column status set data type text
using status::text;

-- Drop the old enum type
drop type public.monitor_check_status;

-- Create the new enum type without 'skipped'
create type public.monitor_check_status as enum (
    'scheduled',
    'checking',
    'failed',
    'success'
);

-- Convert column back to the new enum type
alter table monitor_checks
alter column status set data type public.monitor_check_status
using status::public.monitor_check_status;
