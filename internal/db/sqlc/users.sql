-- name: CreateUser :one
insert into users (email, password_hash, timezone, created_at, updated_at)
values (@email, @password_hash, @timezone, @created_at, @updated_at)
returning *;

-- name: GetUserByEmail :one
select * from users where email = @email limit 1;

-- name: GetUser :one
select * from users where id = @id limit 1;

-- name: UserIntegrations :many
select
    'pushover'::notifier as name,
    exists(
        select 1 from pushover_user_tokens
        where user_id = @user_id
    ) as active
union
select
    'email'::notifier as name,
    true as active;

-- name: UpdateUserTimezone :one
update users
set timezone = @timezone, updated_at = now()
where id = @user_id
returning *;
