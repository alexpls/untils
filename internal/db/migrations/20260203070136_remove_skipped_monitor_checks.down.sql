-- Convert column to text to drop the enum
alter table monitor_checks
alter column status set data type text
using status::text;

-- Drop the new enum type
drop type public.monitor_check_status;

-- Create the old enum type with 'skipped'
create type public.monitor_check_status as enum (
    'scheduled',
    'checking',
    'skipped',
    'failed',
    'success'
);

-- Convert column back to the enum type
alter table monitor_checks
alter column status set data type public.monitor_check_status
using status::public.monitor_check_status;
