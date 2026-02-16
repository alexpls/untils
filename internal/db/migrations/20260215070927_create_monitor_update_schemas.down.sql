alter table monitor_checks
    add column result_id bigint null references monitor_results(id) on delete set null;

create index idx_monitor_checks_result_id on monitor_checks(result_id);
create index idx_monitor_checks_monitor_id_result_id on monitor_checks(monitor_id, result_id);

update monitor_checks mc
set result_id = mr.id
from monitor_results mr
where mc.id = mr.last_confirmed_check_id;

alter table monitor_results
    add column result text null,
    add column date timestamptz null,
    add column date_past_tense_verb text null;

update monitor_results mr
set
    result = coalesce(
        (
            select nullif(f.elem ->> 'value', '')
            from jsonb_array_elements(coalesce(mr.data -> 'fields', '[]'::jsonb)) with ordinality as f(elem, ord)
            where f.elem ->> 'name' = 'Result'
            order by f.ord
            limit 1
        ),
        (
            select nullif(f.elem ->> 'value', '')
            from jsonb_array_elements(coalesce(mr.data -> 'fields', '[]'::jsonb)) with ordinality as f(elem, ord)
            where f.elem ->> 'type' = 'text'
            order by f.ord
            limit 1
        ),
        ''
    ),
    date_past_tense_verb = (
        select nullif(f.elem ->> 'name', '')
        from jsonb_array_elements(coalesce(mr.data -> 'fields', '[]'::jsonb)) with ordinality as f(elem, ord)
        where f.elem ->> 'type' = 'date'
        order by f.ord
        limit 1
    ),
    date = (
        select case
            when nullif(f.elem ->> 'value', '') ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
                then ((f.elem ->> 'value')::date)::timestamptz
            else null
        end
        from jsonb_array_elements(coalesce(mr.data -> 'fields', '[]'::jsonb)) with ordinality as f(elem, ord)
        where f.elem ->> 'type' = 'date'
        order by f.ord
        limit 1
    );

alter table monitor_results alter column result set not null;

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

alter table monitor_results drop column last_confirmed_check_id;
alter table monitor_results drop column last_confirmed_at;
alter table monitor_results drop column data;

drop table if exists monitor_schemas;
