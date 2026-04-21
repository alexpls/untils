-- name: CreateUser :one
insert into users (email, password_hash, timezone, created_at, updated_at)
values (@email, @password_hash, @timezone, @created_at, @updated_at)
returning *;

-- name: GetUserByEmail :one
select * from users where email = @email limit 1;

-- name: GetUser :one
select * from users where id = @id limit 1;

-- name: CountUsers :one
select count(*)::bigint from users;

-- name: UserIntegrations :many
select
    'pushover'::notifier as name,
    exists(
        select 1 from pushover_user_tokens
        where pushover_user_tokens.user_id = @user_id
    ) as configured
union
select
    'email'::notifier as name,
    true as configured
union
select
    'webhook'::notifier as name,
    exists(
        select 1 from webhook_targets
        where webhook_targets.user_id = @user_id
    ) as configured
;

-- name: UpdateUserTimezone :one
update users
set timezone = @timezone, updated_at = now()
where id = @user_id
returning *;

-- name: UpdateUserPasswordHash :one
update users
set password_hash = @password_hash, updated_at = now()
where id = @user_id
returning *;
