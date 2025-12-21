-- name: ListMonitors :many
select * from monitors
where user_id = @user_id and status = 'active'
order by created_at desc;

-- name: GetMonitor :one
select * from monitors
where user_id = @user_id and id = @id;

-- name: CreateMonitor :one
insert into monitors (user_id, subject, instructions, status, updated_at, created_at)
values (@user_id, @subject, @instructions, 'validating', now(), now())
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
set expert = @expert, status = 'ready', updated_at = now()
where user_id = @user_id and id = @monitor_id
returning *;

-- name: UpdateMonitorDraft :one
update monitors
set subject = @subject, instructions = @instructions, updated_at = now()
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

-- name: SkipPendingChecks :exec
update monitor_checks
set status = 'skipped'
where monitor_id = @monitor_id
and status in ('scheduled', 'checking');

-- name: CreateMonitorCheck :one
insert into monitor_checks (monitor_id, status, scheduled_for, done_at)
values (@monitor_id, @status, @scheduled_for, @done_at)
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
set status = 'success', done_at = now()
where id = @id;

-- name: GetLatestMonitorResult :one
select * from monitor_results
where monitor_id = @monitor_id
order by created_at desc
limit 1;

-- name: ListMonitorResults :many
select * from monitor_results
where monitor_id = @monitor_id
order by created_at desc;

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
