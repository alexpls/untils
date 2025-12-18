create extension if not exists pgcrypto;  -- for gen_random_uuid()
create extension if not exists citext;    -- case-insensitive email

create table users (
	id bigserial primary key,

	email citext not null unique,

	password_hash text not null,

	timezone text not null,

	created_at timestamptz not null,
	updated_at timestamptz not null
);

create table sessions(
    id text primary key,
    created_at timestamptz not null,
    expires_at timestamptz not null,
    data jsonb not null
);
create index idx_sessions_expires_at on sessions(expires_at);

create type monitor_status as enum('validating', 'previewing', 'rejected', 'ready', 'active');

create table monitors (
    id bigserial primary key,
    user_id bigint not null references users(id) on delete cascade,
    status monitor_status not null,
    subject text null,
    instructions text null,
    rejected_reason text null,
    updated_at timestamptz not null,
    created_at timestamptz not null
);
create index idx_monitors_user_id_status on monitors(user_id, status);

create type monitor_check_status as enum('scheduled', 'checking', 'skipped', 'failed', 'success');

create table monitor_checks (
    id bigserial primary key,
    monitor_id bigint not null references monitors(id) on delete cascade,
    status monitor_check_status not null,
    scheduled_for timestamptz not null,
    failure_reason text null,
    done_at timestamptz null
);
create index idx_monitor_checks_monitor_id_status_scheduled_for on monitor_checks(monitor_id, status, scheduled_for desc);

create table monitor_results (
    id bigserial primary key,
    monitor_id bigint not null references monitors(id) on delete cascade,
    confirming_check_ids bigint[] not null,
    result text not null,
    date timestamptz null,
    date_past_tense_verb text null,
    citations jsonb not null default '[]',
    latest_confirmation_at timestamptz not null,
    created_at timestamptz not null
);
create index idx_monitor_results_monitor_id on monitor_results(monitor_id, created_at desc);

create table pushover_user_tokens (
    token text primary key,
    user_id bigint not null unique references users(id),
    created_at timestamptz not null
);

create type notifier as enum('email', 'pushover');

create table monitor_notifiers (
    id bigserial primary key,
    monitor_id bigint not null references monitors(id) on delete cascade,
    type notifier not null,
    created_at timestamptz not null
);
create unique index idx_monitor_notifiers_monitor_id_type on monitor_notifiers(monitor_id, type);
