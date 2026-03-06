alter table monitor_results
rename column feedback to correction;

alter table monitor_results
add column hidden boolean not null default false;
