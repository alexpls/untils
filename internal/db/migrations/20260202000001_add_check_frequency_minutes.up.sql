alter table monitors add column check_frequency_minutes integer not null default 1440;
alter table monitors drop column check_schedule;
