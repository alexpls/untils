alter table monitors
add column expert text null;

update monitors
set expert = 'default';
