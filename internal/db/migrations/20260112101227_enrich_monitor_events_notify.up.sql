CREATE OR REPLACE FUNCTION monitor_events_notify()
RETURNS trigger AS $$
DECLARE
  payload_user_id bigint;
  payload_monitor_id bigint;
  rec record;
BEGIN
  IF TG_OP = 'DELETE' THEN
    rec := OLD;
  ELSE
    rec := NEW;
  END IF;

  IF TG_TABLE_NAME = 'monitors' THEN
    payload_monitor_id := rec.id;
    payload_user_id := rec.user_id;
  ELSE
    payload_monitor_id := rec.monitor_id;
    SELECT user_id INTO payload_user_id
    FROM monitors
    WHERE id = payload_monitor_id;
  END IF;

  PERFORM pg_notify(
    'monitor_events',
    json_build_object(
      'table', TG_TABLE_NAME,
      'action', TG_OP,
      'monitor_id', payload_monitor_id,
      'user_id', payload_user_id
    )::text
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
