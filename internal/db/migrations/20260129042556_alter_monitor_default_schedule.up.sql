alter table public.monitors
alter column check_schedule set default '0 8,12,16,20 * * *';

update public.monitors
set check_schedule = '0 8,12,16,20 * * *';
