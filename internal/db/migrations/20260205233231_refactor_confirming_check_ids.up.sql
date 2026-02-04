alter table monitor_checks
    add column result_id bigint null references monitor_results(id) on delete set null;

create index idx_monitor_checks_result_id on monitor_checks(result_id);
create index idx_monitor_checks_monitor_id_result_id on monitor_checks(monitor_id, result_id);

update monitor_checks mc
set result_id = mr.id
from monitor_results mr
where mc.id = any(mr.confirming_check_ids);

create view monitor_results_with_latest_check as
with latest_checks as (
    select distinct on (result_id)
        result_id,
        id as latest_check_id,
        done_at as latest_confirmation_at
    from monitor_checks
    where status = 'success'
    and result_id is not null
    order by result_id, done_at desc
)
select
    mr.id,
    mr.monitor_id,
    mr.result,
    mr.date,
    mr.date_past_tense_verb,
    mr.citations,
    mr.created_at,
    mr.feedback,
    lc.latest_confirmation_at,
    lc.latest_check_id
from monitor_results mr
left join latest_checks lc on lc.result_id = mr.id;

-- drop the old columns that are now redundant
alter table monitor_results
    drop column if exists confirming_check_ids,
    drop column if exists latest_confirmation_at;
