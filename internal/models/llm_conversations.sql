-- name: CreateLLMConversation :one
insert into llm_conversations (user_id, source_type, source_id, created_at, updated_at)
values (@user_id, @source_type, @source_id, now(), now())
returning *;

-- name: AddMessageToLLMConversation :exec
update llm_conversations
set messages = messages || @message, updated_at = now()
where id = @llm_conversation_id;

-- name: GetLLMConversationBySourceID :one
select * from llm_conversations
where source_type = @source_type and source_id = @source_id
order by created_at desc
limit 1;

-- name: GetTimelineEventsBySourceID :many
with assistant_messages as (
  select m
  from llm_conversations c,
      jsonb_array_elements(c.messages) as m
  where c.source_type = @source_type
    and c.source_id = @source_id
    and m->>'role' = 'assistant'
)
select
    (tc->>'name')::text as name,
    (tc->>'arguments')::text as arguments,
    (m->>'at')::timestamptz as at
from assistant_messages,
    jsonb_array_elements(coalesce(m->'body'->'tool_calls', '[]'::jsonb)) as tc
order by (m->>'at')::timestamptz;
