create table webhook_targets (
    id bigserial primary key,
    user_id bigint not null references users (id) on delete cascade,
    url text,
    created_at timestamptz not null default now()
);

create index idx_webhook_targets_user_id on webhook_targets (user_id);

alter type notifier add value 'webhook';
