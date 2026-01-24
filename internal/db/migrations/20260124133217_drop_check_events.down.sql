create type monitor_check_event_kind as enum (
    'web_search',
    'browser_navigate',
    'browser_click',
    'browser_wait'
);

create table monitor_check_events (
    id bigserial primary key,
    monitor_id bigint not null references monitors(id) on delete cascade,
    monitor_check_id bigint not null references monitor_checks(id) on delete cascade,
    kind monitor_check_event_kind not null,
    details jsonb not null,
    created_at timestamp with time zone default now() not null
);

create index idx_monitor_check_events_monitor_check_id on monitor_check_events using btree (monitor_check_id);

create trigger monitor_check_events_notify_trigger after insert or delete or update on monitor_check_events for each row execute function monitor_events_notify();
