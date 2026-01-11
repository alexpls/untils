-- name: GetPushoverUserToken :one
select * from pushover_user_tokens
where user_id = @user_id;

-- name: CreateOrUpdatePushoverUserToken :one
insert into pushover_user_tokens (token, user_id, created_at)
values (@token, @user_id, now())
on conflict (user_id) do update
set token = excluded.token, created_at = now()
returning *;

-- name: DeletePushoverUserToken :exec
delete from pushover_user_tokens
where user_id = @user_id;
