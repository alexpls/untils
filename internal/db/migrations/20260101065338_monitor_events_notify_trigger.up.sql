create or replace function monitor_events_notify()
returns trigger as $$
begin
  perform pg_notify(
    'monitor_events',
    json_build_object(
      'table', TG_TABLE_NAME,
      'action', TG_OP,
      'data', row_to_json(coalesce(NEW, OLD))
    )::text
  );
  return NEW;
end;
$$ language plpgsql;

create or replace trigger monitors_notify_trigger
after insert or update or delete on monitors
for each row execute function monitor_events_notify();

create or replace trigger monitor_checks_notify_trigger
after insert or update or delete on monitor_checks
for each row execute function monitor_events_notify();

create or replace trigger monitor_results_notify_trigger
after insert or update or delete on monitor_results
for each row execute function monitor_events_notify();

create or replace trigger monitor_check_events_notify_trigger
after insert or update or delete on monitor_check_events
for each row execute function monitor_events_notify();
