-- name: CreateEmailSubscriber :exec
insert into email_subscribers (email)
values (@email)
on conflict (email) do nothing;
