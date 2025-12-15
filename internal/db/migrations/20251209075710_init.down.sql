drop type notifier;
drop table monitor_notifiers;
drop table pushover_user_tokens;
drop table monitor_results;
drop table monitor_checks;
drop table monitors;
drop type monitor_check_status;
drop type monitor_status;
drop index idx_sessions_expires_at;
drop table sessions;
drop table users;

drop extension if exists pgcrypto;
drop extension if exists citext;
