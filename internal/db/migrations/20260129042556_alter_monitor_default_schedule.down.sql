alter table public.monitors
alter column check_schedule set default '0 */6 * * *';
