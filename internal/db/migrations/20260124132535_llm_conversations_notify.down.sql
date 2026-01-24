drop trigger if exists llm_conversations_notify_trigger on llm_conversations;

create or replace function monitor_events_notify()
returns trigger as $$
declare
  payload_user_id bigint;
  payload_monitor_id bigint;
  rec record;
begin
  if tg_op = 'DELETE' then
    rec := OLD;
  else
    rec := NEW;
  end if;

  if tg_table_name = 'monitors' then
    payload_monitor_id := rec.id;
    payload_user_id := rec.user_id;
  else
    payload_monitor_id := rec.monitor_id;
    select user_id into payload_user_id
    from monitors
    where id = payload_monitor_id;
  end if;

  perform pg_notify(
    'monitor_events',
    json_build_object(
      'table', tg_table_name,
      'action', tg_op,
      'monitor_id', payload_monitor_id,
      'user_id', payload_user_id
    )::text
  );
  return NEW;
end;
$$ language plpgsql;
