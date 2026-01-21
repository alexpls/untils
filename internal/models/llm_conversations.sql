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
