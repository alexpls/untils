create table api_tokens (
    id bigserial primary key,
    key_hash text not null,
    user_id bigint not null references users (id) on delete cascade,
    name text not null,
    last_used_at timestamptz,
    created_at timestamptz not null default now()
);

create unique index idx_api_tokens_key_hash on api_tokens (key_hash);
create index idx_api_tokens_user_id_created_at on api_tokens (user_id, created_at desc);
