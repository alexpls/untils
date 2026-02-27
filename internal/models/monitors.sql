-- name: ListMonitorsWithResults :many
with next_check as (
    select distinct on (monitor_id) monitor_id, scheduled_for
    from monitor_checks
    where status in ('scheduled')
    order by monitor_id, scheduled_for desc
),
current_check as (
    select distinct on (monitor_id) monitor_id
    from monitor_checks
    where status = 'checking'
    order by monitor_id, scheduled_for desc
)
select
    m.id as monitor_id,
    m.status,
    m.subject::text as subject,
    m.check_frequency_minutes,
    m.created_at,
    coalesce(mr.headline::text, '')::text as headline,
    coalesce(mr.subtitle::text, '')::text as subtitle,
    coalesce(mr.data, '{"fields": []}'::jsonb) as data,
    coalesce(mr.created_at, '0001-01-01 00:00:00 +0000') as latest_result_created_at,
    coalesce(mc.scheduled_for, '0001-01-01 00:00:00 +0000') as next_check_scheduled_for,
    (cc.monitor_id is not null)::boolean as currently_checking
from monitors m
-- using a subquery instead of a cte so sqlc can grab a reference to the monitor_results.data
-- type and override it via config. this doesn't work with ctes.
-- https://github.com/sqlc-dev/sqlc/issues/3438 should fix this.
left join (
    select distinct on (monitor_id) monitor_id, headline, subtitle, data, created_at
    from monitor_results
    order by monitor_id, created_at desc, id desc
) mr on mr.monitor_id = m.id
left join next_check mc on mc.monitor_id = m.id
left join current_check cc on cc.monitor_id = m.id
where m.user_id = @user_id
order by mr.created_at desc
limit @page_size offset @row_offset;

-- name: GetMonitor :one
select * from monitors
where user_id = @user_id and id = @id;

-- name: CreateMonitor :one
insert into monitors (user_id, subject, status, updated_at, created_at)
values (@user_id, @subject, 'validating', now(), now())
returning *;

-- name: DeleteMonitor :exec
delete from monitors
where user_id = @user_id and id = @monitor_id;

-- name: RejectMonitor :exec
update monitors
set status = 'rejected',
    rejected_reason = @rejected_reason,
    updated_at = now()
where user_id = @user_id and id = @id;

-- name: UpdateMonitorStatus :one
update monitors
set status = @status, updated_at = now()
where user_id = @user_id and id = @id
returning *;

-- name: UpdateMonitorToReady :one
update monitors
set status = 'ready', subject = @subject, updated_at = now()
where user_id = @user_id and id = @monitor_id
returning *;

-- name: UpdateMonitorDraft :one
update monitors
set subject = @subject, updated_at = now()
where user_id = @user_id and id = @id and status != 'active'
returning *;

-- name: UpdateMonitorCheckFrequency :one
update monitors
set check_frequency_minutes = @check_frequency_minutes, updated_at = now()
where id = @monitor_id
returning *;

-- name: UpdateMonitorToggleAutoActivate :one
update monitors
set auto_activate = not auto_activate
where id = @monitor_id
returning *;

-- name: GetMonitorCheck :one
select * from monitor_checks
where id = @id;

-- name: GetNextMonitorCheck :one
select * from monitor_checks
where monitor_id = @monitor_id
and (status = 'scheduled' or status = 'checking')
order by scheduled_for desc
limit 1;

-- name: GetInProgressMonitorCheck :one
select * from monitor_checks
where monitor_id = @monitor_id
and status = 'checking'
order by scheduled_for desc
limit 1;

-- name: GetMonitorCheckStats :one
select
    count(*) filter (where mc.done_at >= now() - interval '7 days') as checks_last_7_days,
    count(*) filter (where mc.done_at >= now() - interval '30 days') as checks_last_30_days,
    count(*) as checks_all_time
from monitor_checks mc
join monitors m on m.id = mc.monitor_id
where m.user_id = @user_id
and mc.status = 'success';

-- name: GetDailyMonitorCheckCounts :many
select
    cast(calendar.day as date) as day,
    count(mc.id)::int as check_count
from generate_series(now() - interval '6 days', now(), '1 day') as calendar(day)
left join monitors m on m.user_id = @user_id
left join monitor_checks mc on mc.monitor_id = m.id
    and mc.status = 'success'
    and cast(mc.done_at as date) = cast(calendar.day as date)
group by calendar.day
order by day asc;

-- name: DeleteScheduledChecks :exec
delete from monitor_checks
where monitor_id = @monitor_id
and status in ('scheduled');

-- name: DeleteStaleChecks :exec
delete from monitor_checks
where monitor_id = @monitor_id
and status in ('scheduled', 'checking');

-- name: CreateMonitorCheck :one
insert into monitor_checks (monitor_id, status, scheduled_for, done_at, result)
values (@monitor_id, @status, @scheduled_for, @done_at, @result)
returning *;

-- name: UpdateMonitorCheckChecking :exec
update monitor_checks
set status = 'checking'
where id = @id;

-- name: UpdateMonitorCheckFailed :exec
update monitor_checks
set status = 'failed', failure_reason = @failure_reason, done_at = now()
where id = @id;

-- name: UpdateMonitorCheckSuccess :exec
update monitor_checks
set status = 'success', result = @result, done_at = now()
where id = @id;

-- name: GetLatestMonitorResult :one
select * from monitor_results
where monitor_id = @monitor_id
order by created_at desc, id desc
limit 1;

-- name: GetPreviousResultsWithCheck :many
select sqlc.embed(mr), sqlc.embed(mc)
from monitor_results mr
join lateral (
    select mc.*
    from monitor_result_checks mrc
    join monitor_checks mc on mc.id = mrc.monitor_check_id
    where mrc.monitor_result_id = mr.id
    order by mc.done_at desc nulls last, mc.id desc
    limit 1
) mc on true
where mr.monitor_id = @monitor_id
order by mr.created_at desc, mr.id desc
limit 10;

-- name: ListMonitorResults :many
select * from monitor_results
where monitor_id = @monitor_id
order by created_at desc, id desc;

-- name: ListMonitorResultsWithLatestCheck :many
select
    sqlc.embed(mr),
    mc.id as latest_check_id,
    mc.done_at as latest_check_done_at
from monitor_results mr
join lateral (
    select mc.id, mc.done_at
    from monitor_result_checks mrc
    join monitor_checks mc on mc.id = mrc.monitor_check_id
    where mrc.monitor_result_id = mr.id
    and mc.done_at is not null
    order by mc.done_at desc nulls last, mc.id desc
    limit 1
) mc on true
where mr.monitor_id = @monitor_id
order by mr.created_at desc, mr.id desc;

-- name: GetMonitorResult :one
select * from monitor_results
where monitor_id = @monitor_id
and id = @result_id;

-- name: ListMonitorResultsByCheckID :many
select mr.*
from monitor_results mr
join monitor_result_checks mrc on mrc.monitor_result_id = mr.id
where mrc.monitor_check_id = @check_id
order by mr.created_at desc, mr.id desc
;

-- name: DeleteMonitorChecks :exec
delete from monitor_checks
where monitor_id = @monitor_id;

-- name: DeleteMonitorResults :exec
delete from monitor_results
where monitor_id = @monitor_id;

-- name: CreateMonitorResult :one
insert into monitor_results (
    monitor_id,
    headline,
    subtitle,
    data,
    citations,
    created_at
)
values (
    @monitor_id,
    @headline,
    @subtitle,
    @data,
    @citations,
    now()
)
returning *;

-- name: CreateMonitorResultCheck :exec
insert into monitor_result_checks (monitor_result_id, monitor_check_id)
values (@monitor_result_id, @monitor_check_id)
on conflict (monitor_result_id, monitor_check_id) do nothing;

-- name: UpdateMonitorResultWithFeedback :exec
update monitor_results
set feedback = @feedback
where id = @monitor_result_id;

-- name: ListMonitorActivity :many
select
    mon.id::bigint as monitor_id,
    res.id::bigint as result_id,
    mon.subject,
    res.headline,
    res.created_at,
    res.subtitle,
    res.data
from monitor_results res
left join monitors mon on mon.id = res.monitor_id
where mon.status = 'active'
and mon.user_id = @user_id
order by res.created_at desc
limit 7;

-- name: GetMonitorSchema :one
select * from monitor_schemas
where monitor_id = @monitor_id;

-- name: UpsertMonitorSchema :one
insert into monitor_schemas (monitor_id, data, created_at)
values (@monitor_id, @data, now())
on conflict (monitor_id) do update
set data = excluded.data, created_at = now()
returning *;

-- name: DeleteMonitorSchema :exec
delete from monitor_schemas
where monitor_id = @monitor_id;

-- name: BumpMonitorVersion :exec
update monitors set updated_at = now() where id = @monitor_id;

-- name: CreateMonitorNotifier :one
insert into monitor_notifiers (monitor_id, type, created_at)
values (@monitor_id, @type, now())
returning *;

-- name: DeleteMonitorNotifier :exec
delete from monitor_notifiers
where monitor_id = @monitor_id and type = @type;

-- name: ListMonitorNotifiers :many
select * from monitor_notifiers
where monitor_id = @monitor_id
order by created_at desc;

-- name: DeleteMonitorNotifiersByUserAndType :exec
delete from monitor_notifiers
where type = @type
and monitor_id in (select id from monitors where user_id = @user_id);

-- name: ListMonitorChecks :many
select * from monitor_checks
where monitor_id = @monitor_id
order by scheduled_for desc
limit 30;

-- name: ListChecksWithMonitor :many
select
    mc.id as check_id,
    mc.monitor_id,
    mc.status,
    mc.scheduled_for,
    mc.done_at,
    m.subject::text as monitor_subject
from monitor_checks mc
join monitors m on m.id = mc.monitor_id
where m.user_id = @user_id
order by mc.scheduled_for desc
limit @page_size offset @row_offset;

-- name: GetCheckWithMonitor :one
select
    mc.id as check_id,
    mc.monitor_id,
    mc.status,
    mc.scheduled_for,
    mc.done_at,
    mc.failure_reason,
    mc.result,
    m.subject::text as monitor_subject,
    m.user_id
from monitor_checks mc
join monitors m on m.id = mc.monitor_id
where mc.id = @check_id;

-- name: GetLastScheduledCheckTime :one
select scheduled_for from monitor_checks
where monitor_id = @monitor_id
order by scheduled_for desc
limit 1;

-- name: UpdateMonitorCheckScheduledFor :exec
update monitor_checks
set scheduled_for = @scheduled_for
where id = @id;


-- name: RescheduleRiverJobNow :exec
update river_job
set scheduled_at = now(), state = 'available'
where kind = 'check'
and args->>'monitor_check_id' = @check_id::text
and state = 'scheduled';

-- name: ListCheckRiverJobIDsByMonitorID :many
select rj.id
from river_job rj
join monitor_checks mc on rj.args->>'monitor_check_id' = mc.id::text
where rj.kind = 'check'
and rj.finalized_at is null
and mc.monitor_id = @monitor_id
order by rj.id asc;

-- name: ListValidateDraftRiverJobIDsByMonitorID :many
select id
from river_job
where kind = 'validate_draft'
and finalized_at is null
and args->>'monitor_id' = @monitor_id::text
order by id asc;
