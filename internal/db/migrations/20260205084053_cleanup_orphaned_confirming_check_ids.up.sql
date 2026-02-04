-- remove orphaned check ids from confirming_check_ids arrays.
-- these were created by a race condition where checks were marked 'skipped'
-- while still running, then later deleted by migration 20260203070136.

-- for results where some check ids are valid, keep only valid ones
update monitor_results mr
set confirming_check_ids = (
    select array_agg(check_id)
    from unnest(mr.confirming_check_ids) as check_id
    where exists (select 1 from monitor_checks where id = check_id)
)
where exists (
    -- has at least one orphaned check id
    select 1
    from unnest(mr.confirming_check_ids) as check_id
    where not exists (select 1 from monitor_checks where id = check_id)
)
and exists (
    -- has at least one valid check id
    select 1
    from unnest(mr.confirming_check_ids) as check_id
    where exists (select 1 from monitor_checks where id = check_id)
);

-- for results where all check ids are orphaned, use the most recent
-- successful check for that monitor before the result's confirmation time
update monitor_results mr
set confirming_check_ids = array[(
    select mc.id 
    from monitor_checks mc 
    where mc.monitor_id = mr.monitor_id 
    and mc.status = 'success' 
    and mc.done_at <= mr.latest_confirmation_at 
    order by mc.done_at desc 
    limit 1
)]
where not exists (
    -- no valid check ids remain
    select 1
    from unnest(mr.confirming_check_ids) as check_id
    where exists (select 1 from monitor_checks where id = check_id)
);
