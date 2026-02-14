create or replace function monitor_events_notify()
returns trigger as $$
declare
  payload_user_id bigint;
  payload_monitor_id bigint;
  rec record;
begin
  if tg_op = 'DELETE' then
    rec := old;
  else
    rec := new;
  end if;

  if tg_table_name = 'monitors' then
    payload_monitor_id := rec.id;
    payload_user_id := rec.user_id;
  elsif tg_table_name = 'llm_conversations' then
    if rec.source_type = 'check' then
      payload_user_id := rec.user_id;
      select monitor_id into payload_monitor_id
      from monitor_checks
      where id = rec.source_id;
    end if;
  else
    payload_monitor_id := rec.monitor_id;
    select user_id into payload_user_id
    from monitors
    where id = payload_monitor_id;
  end if;

  if payload_monitor_id is not null then
    perform pg_notify(
      'monitor_events',
      json_build_object(
        'table', tg_table_name,
        'action', tg_op,
        'monitor_id', payload_monitor_id,
        'user_id', payload_user_id
      )::text
    );
  end if;
  return new;
end;
$$ language plpgsql;
