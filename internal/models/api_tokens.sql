-- name: CreateAPIToken :one
insert into api_tokens (key_hash, user_id, name, created_at)
values (@key_hash, @user_id, @name, now())
returning *;

-- name: ListAPITokens :many
select * from api_tokens
where user_id = @user_id
order by created_at desc;

-- name: GetAPITokenByKeyHash :one
select * from api_tokens
where key_hash = @key_hash;

-- name: DeleteAPIToken :execrows
delete from api_tokens
where user_id = @user_id
and id = @id;

-- name: UpdateAPITokenLastUsedAt :exec
update api_tokens
set last_used_at = now()
where id = @id
and (last_used_at is null or last_used_at < now());
