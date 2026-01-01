create type monitor_check_event_kind as enum (
    'web_search',
    'browser_navigate',
    'browser_click'
);

create table monitor_check_events (
    id bigserial primary key,
    monitor_check_id bigint not null references monitor_checks(id) on delete cascade,
    kind monitor_check_event_kind not null,
    details jsonb not null,
    created_at timestamptz not null default now()
);
create index idx_monitor_check_events_monitor_check_id on monitor_check_events(monitor_check_id);
