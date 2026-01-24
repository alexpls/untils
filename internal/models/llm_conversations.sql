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
where source_type = @source_type and source_id = @source_id;

-- name: GetTimelineEventsBySourceID :many
select
    (out->>'name')::text as name,
    (out->>'arguments')::text as arguments,
    (m->>'at')::timestamptz as at
from llm_conversations c,
    jsonb_array_elements(c.messages) as m,
    jsonb_array_elements(m->'body'->'output') as out
where c.source_type = @source_type
  and c.source_id = @source_id
  and m->>'role' = 'assistant'
  and out->>'type' = 'function_call'
order by (m->>'at')::timestamptz;
