alter table monitor_results
drop column hidden;

alter table monitor_results
rename column correction to feedback;
