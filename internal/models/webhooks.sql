-- name: ListWebhookTargets :many
select * from webhook_targets
where user_id = @user_id
order by created_at desc;

-- name: CreateWebhookTarget :exec
insert into webhook_targets (user_id, url, created_at)
values (@user_id, @url, now());

-- name: GetWebhookTargetByURL :one
select * from webhook_targets
where user_id = @user_id
and url = @url;

-- name: GetWebhookTarget :one
select * from webhook_targets
where user_id = @user_id
and id = @webhook_target_id;

-- name: DeleteWebhookTarget :exec
delete from webhook_targets
where user_id = @user_id
and id = @webhook_target_id;
