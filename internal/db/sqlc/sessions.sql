-- name: GetSession :one
select * from sessions
where id = @id
and expires_at > now();

-- name: SaveSession :exec
insert into sessions (id, created_at, expires_at, data)
values (@id, @created_at, @expires_at, @data)
on conflict (id) do update
set expires_at = excluded.expires_at, data = excluded.data;

-- name: DestroySession :exec
delete from sessions
where id = @id;

-- name: TrimExpiredSessions :execrows
delete from sessions
where expires_at <= now();
