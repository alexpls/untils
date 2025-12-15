-- name: CreateUser :one
insert into users (email, password_hash, timezone, created_at, updated_at)
values (@email, @password_hash, @timezone, @created_at, @updated_at)
returning *;

-- name: GetUserByEmail :one
select * from users where email = @email limit 1;

-- name: GetUser :one
select * from users where id = @id limit 1;

-- name: ActiveUserIntegrations :one
select
    exists(
        select 1 from pushover_user_tokens
        where user_id = @user_id
    ) as pushover_active,
     -- hack to make sqlc generate a struct for this instead of returning single value.
     -- remove after adding a second value.
    false as placeholder;
