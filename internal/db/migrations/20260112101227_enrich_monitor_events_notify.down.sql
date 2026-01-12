CREATE OR REPLACE FUNCTION monitor_events_notify()
RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify(
    'monitor_events',
    json_build_object(
      'table', TG_TABLE_NAME,
      'action', TG_OP,
      'data', row_to_json(coalesce(NEW, OLD))
    )::text
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
