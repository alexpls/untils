-- add back the old columns (data is lost, but structure is restored)
alter table monitor_results
    add column if not exists confirming_check_ids bigint[] default '{}',
    add column if not exists latest_confirmation_at timestamp with time zone;

drop view if exists monitor_results_with_latest_check;

drop index if exists idx_monitor_checks_monitor_id_result_id;
drop index if exists idx_monitor_checks_result_id;

alter table monitor_checks drop column if exists result_id;
