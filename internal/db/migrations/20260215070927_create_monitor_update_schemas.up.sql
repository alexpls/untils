create table monitor_schemas (
    id bigserial primary key,
    monitor_id bigint not null references monitors (id) on delete cascade,
    data jsonb not null,
    created_at timestamptz not null default now()
);

create unique index idx_monitor_schemas_monitor_id on monitor_schemas(monitor_id);

create table monitor_result_checks (
    monitor_result_id bigint not null references monitor_results(id) on delete cascade,
    monitor_check_id bigint not null references monitor_checks(id) on delete cascade,
    primary key (monitor_result_id, monitor_check_id)
);

create index idx_monitor_result_checks_check_id on monitor_result_checks(monitor_check_id);

insert into monitor_result_checks (monitor_result_id, monitor_check_id)
select
    mr.id,
    mc.id
from monitor_results mr
join monitor_checks mc on mc.result_id = mr.id
on conflict (monitor_result_id, monitor_check_id) do nothing;

with unlinked_results as (
    select mr.id, mr.monitor_id, mr.created_at
    from monitor_results mr
    where not exists (
        select 1
        from monitor_result_checks mrc
        where mrc.monitor_result_id = mr.id
    )
),
candidate_checks as (
    select
        ur.id as monitor_result_id,
        mc.id as monitor_check_id,
        mc.done_at,
        case
            when mc.status = 'success' and mc.done_at <= ur.created_at then 1
            when mc.status = 'success' then 2
            when mc.done_at <= ur.created_at then 3
            else 4
        end as priority
    from unlinked_results ur
    join monitor_checks mc on mc.monitor_id = ur.monitor_id
),
ranked_checks as (
    select
        monitor_result_id,
        monitor_check_id,
        done_at,
        row_number() over (
            partition by monitor_result_id
            order by
                priority asc,
                done_at desc nulls last,
                monitor_check_id desc
        ) as rn
    from candidate_checks
)
insert into monitor_result_checks (monitor_result_id, monitor_check_id)
select
    rc.monitor_result_id,
    rc.monitor_check_id
from ranked_checks rc
where rc.rn = 1
on conflict (monitor_result_id, monitor_check_id) do nothing;

alter table monitor_results add column last_confirmed_check_id bigint references monitor_checks(id);
alter table monitor_results add column last_confirmed_at timestamptz;

with latest_checks as (
    select distinct on (mrc.monitor_result_id)
        mrc.monitor_result_id as result_id,
        mc.id as check_id,
        coalesce(mc.done_at, mr.created_at) as confirmed_at
    from monitor_result_checks mrc
    join monitor_checks mc on mc.id = mrc.monitor_check_id
    join monitor_results mr on mr.id = mrc.monitor_result_id
    order by
        mrc.monitor_result_id,
        coalesce(mc.done_at, mr.created_at) desc,
        mc.id desc
)
update monitor_results mr
set
    last_confirmed_check_id = lc.check_id,
    last_confirmed_at = lc.confirmed_at
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
