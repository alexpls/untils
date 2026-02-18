create table monitor_schemas (
    id bigserial primary key,
	monitor_id bigint not null references monitors (id) on delete cascade,
	data jsonb not null,
	created_at timestamptz not null default now()
);

create unique index idx_monitor_schemas_monitor_id on monitor_schemas(monitor_id);

-- add last_confirmed* denormalized cols

alter table monitor_results add column last_confirmed_check_id bigint references monitor_checks(id);
alter table monitor_results add column last_confirmed_at timestamptz;

with candidate_checks as (
    select
        mr.id as result_id,
        mc.id as check_id,
        mc.done_at,
        0 as priority
    from monitor_results mr
    join monitor_checks mc on mc.result_id = mr.id

    union all

    -- fallback for legacy rows with missing result_id links
    select
        mr.id as result_id,
        mc.id as check_id,
        mc.done_at,
        case
            when mc.status = 'success' and mc.done_at <= mr.created_at then 1
            when mc.status = 'success' then 2
            when mc.done_at <= mr.created_at then 3
            else 4
        end as priority
    from monitor_results mr
    join monitor_checks mc on mc.monitor_id = mr.monitor_id
    where mc.result_id is distinct from mr.id
),
ranked_checks as (
    select
        result_id,
        check_id,
        done_at,
        row_number() over (
            partition by result_id
            order by
                priority asc,
                done_at desc nulls last,
                check_id desc
        ) as rn
    from candidate_checks
),
latest_checks as (
    select result_id, check_id, done_at
    from ranked_checks
    where rn = 1
)
update monitor_results mr
set
    last_confirmed_check_id = lc.check_id,
    last_confirmed_at = coalesce(lc.done_at, mr.created_at)
from latest_checks lc
where lc.result_id = mr.id;

alter table monitor_results alter column last_confirmed_check_id set not null;
alter table monitor_results alter column last_confirmed_at set not null;

drop view if exists monitor_results_with_latest_check;

alter table monitor_checks drop column result_id;

-- add new data col and denormalized

alter table monitor_results add column headline text;
alter table monitor_results add column subtitle text not null default '';
alter table monitor_results add column data jsonb not null default '{}'::jsonb;

-- migrate data

with monitor_settings as (
    select
        mon.monitor_id,
        (
            select nullif(btrim(mr2.date_past_tense_verb), '')
            from monitor_results mr2
            where mr2.monitor_id = mon.monitor_id
                and mr2.date is not null
                and nullif(btrim(mr2.date_past_tense_verb), '') is not null
            order by mr2.created_at desc, mr2.id desc
            limit 1
        ) as date_field_name
    from (
        select distinct monitor_id
        from monitor_results
    ) mon
)
update monitor_results mr
set
    headline = '{{Result}}',
    subtitle = case
        when ms.date_field_name is not null then ms.date_field_name || ': {{' || ms.date_field_name || '}}'
        else ''
    end,
    data = jsonb_build_object(
    'fields',
    jsonb_build_array(
        jsonb_build_object(
            'type', 'text',
            'name', 'Result',
            'value', mr.result
        )
    ) || case
        when ms.date_field_name is not null then jsonb_build_array(
            jsonb_build_object(
                'type', 'date',
                'name', ms.date_field_name,
                'value', case
                    when mr.date is null then ''
                    else to_char(mr.date at time zone 'utc', 'yyyy-mm-dd')
                end
            )
        )
        else '[]'::jsonb
    end
)
from monitor_settings ms
where ms.monitor_id = mr.monitor_id;

alter table monitor_results alter column headline set not null;

with monitor_settings as (
    select
        mon.monitor_id,
        (
            select nullif(btrim(mr2.date_past_tense_verb), '')
            from monitor_results mr2
            where mr2.monitor_id = mon.monitor_id
                and mr2.date is not null
                and nullif(btrim(mr2.date_past_tense_verb), '') is not null
            order by mr2.created_at desc, mr2.id desc
            limit 1
        ) as date_field_name
    from (
        select distinct monitor_id
        from monitor_results
    ) mon
)
insert into monitor_schemas (monitor_id, data)
select
    ms.monitor_id,
    jsonb_build_object(
        'fields',
        jsonb_build_array(
            jsonb_build_object(
                'type', 'text',
                'name', 'Result'
            )
        ) || case
            when ms.date_field_name is not null then jsonb_build_array(
                jsonb_build_object(
                    'type', 'date',
                    'name', ms.date_field_name
                )
            )
            else '[]'::jsonb
        end
    )
from monitor_settings ms;

alter table monitor_results drop column result;
alter table monitor_results drop column date;
alter table monitor_results drop column date_past_tense_verb;
alter table monitor_results alter column data drop default;
