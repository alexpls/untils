create type llm_conversations_source as enum('check');

create table llm_conversations (
    id bigserial primary key,
    user_id bigint not null references users(id) on delete cascade,
    source_type llm_conversations_source not null,
    source_id bigint not null,
    messages jsonb not null default '[]',
    created_at timestamp not null,
    updated_at timestamp not null
);

create index idx_llm_conversations_user_id_source_type_source_id
    on llm_conversations (user_id, source_type, source_id);
