-- name: ListMonitors :many
select * from monitors
where user_id = @user_id and status = 'active'
order by created_at desc;

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

-- name: SkipPendingChecks :exec
update monitor_checks
set status = 'skipped'
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
order by created_at desc
limit 1;

-- name: GetPreviousResultsWithCheck :many
select sqlc.embed(mr), sqlc.embed(mc)
from monitor_results mr
inner join monitor_checks mc on mc.id = mr.confirming_check_ids[array_length(mr.confirming_check_ids, 1)]
where mr.monitor_id = @monitor_id
order by mr.created_at desc
limit 10;

-- name: ListMonitorResults :many
select * from monitor_results
where monitor_id = @monitor_id
order by created_at desc;

-- name: GetMonitorResult :one
select * from monitor_results
where monitor_id = @monitor_id
and id = @result_id;

-- name: DeleteMonitorChecks :exec
delete from monitor_checks
where monitor_id = @monitor_id;

-- name: DeleteMonitorResults :exec
delete from monitor_results
where monitor_id = @monitor_id;

-- name: CreateMonitorResult :one
insert into monitor_results (monitor_id, confirming_check_ids, result, date, date_past_tense_verb, citations, latest_confirmation_at, created_at)
values (@monitor_id, @confirming_check_ids, @result, @date, @date_past_tense_verb, @citations, now(), now())
returning *;

-- name: AppendConfirmingCheckIDToResult :exec
update monitor_results
set confirming_check_ids = array_append(confirming_check_ids, @confirming_check_id_to_append::bigint),
    latest_confirmation_at = now()
where id = @monitor_result_id;

-- name: UpdateMonitorResultWithFeedback :exec
update monitor_results
set feedback = @feedback
where id = @monitor_result_id;

-- name: ListMonitorActivity :many
select
    mon.id::bigint as monitor_id,
    res.id::bigint as result_id,
    mon.subject,
    res.result,
    res.created_at,
    res.date,
    res.date_past_tense_verb
from monitor_results res
left join monitors mon on mon.id = res.monitor_id
where mon.status = 'active'
and mon.user_id = @user_id
order by res.created_at desc
limit 7;

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

-- name: ListMonitorCheckEvents :many
select * from monitor_check_events
where monitor_check_id = @monitor_check_id
order by created_at asc;

-- name: CreateMonitorCheckEvent :one
insert into monitor_check_events (monitor_id, monitor_check_id, kind, details, created_at)
values (@monitor_id, @monitor_check_id, @kind, @details, now())
returning *;

-- name: DeleteMonitorCheckEventsForMonitor :exec
delete from monitor_check_events
where monitor_id = @monitor_id;
