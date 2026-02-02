alter table monitors add column check_schedule text not null default '0 8,12,16,20 * * *';
alter table monitors drop column check_frequency_minutes;
