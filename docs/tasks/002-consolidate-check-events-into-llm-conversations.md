# Consolidate check_events into llm_conversations

## Background

Currently we have two data models that track LLM workflow activity during a monitor check:

1. **`monitor_check_events`** - Stores discrete user-facing actions (web_search, browser_navigate, browser_click, browser_wait)
2. **`llm_conversations`** - Stores the full LLM conversation (system, user, assistant, tool messages)

These have significant overlap - both record tool calls, just at different granularities. The check_events table was designed for real-time UI updates, while llm_conversations was designed for debugging.

## Goal

Consolidate onto a single data model (`llm_conversations`) to simplify the codebase and have a single source of truth for check activity.

## Key Insight

The `assistant` message already contains tool call information when the LLM requests tools. The message body includes:

```json
{
  "output": [
    {
      "type": "function_call",
      "call_id": "...",
      "name": "browser_navigate",
      "arguments": "{\"url\": \"...\"}"
    }
  ]
}
```

We can parse this in the UI layer to display the same "in progress" timeline we currently show with check_events.

### Tool Display Rendering

The current UI renders tool events with context-specific details, e.g.:
- `Searching for "taylor swift albums"`
- `Browsing "wikipedia.org"`
- `Waiting for the page to load`
- `Clicking on a link`

The `arguments` field in the function call contains the parameters needed to render these details (e.g. `{"url": "..."}` or `{"query": "..."}`). 

Display logic is kept in the UI layer via a simple switch statement in the template package (`internal/monitor/check_view.templ`). This keeps display concerns out of the service/model layer and makes it easy to add new tool display formats in one place.

## Implementation Plan

### Phase 1: Add pg_notify trigger on llm_conversations

Update the existing `monitor_events_notify()` function to handle `llm_conversations`. Since this table doesn't have a `monitor_id` column directly, we need to look it up via `monitor_checks` when `source_type = 'check'`.

Create a new migration:

```sql
CREATE OR REPLACE FUNCTION monitor_events_notify() RETURNS TRIGGER AS $$
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
  ELSIF TG_TABLE_NAME = 'llm_conversations' THEN
    -- Only emit for check source type
    IF rec.source_type = 'check' THEN
      payload_user_id := rec.user_id;
      -- Look up monitor_id from monitor_checks via source_id
      SELECT monitor_id INTO payload_monitor_id
      FROM monitor_checks
      WHERE id = rec.source_id;
    END IF;
  ELSE
    payload_monitor_id := rec.monitor_id;
    SELECT user_id INTO payload_user_id
    FROM monitors
    WHERE id = payload_monitor_id;
  END IF;

  -- Only notify if we have a valid monitor_id
  IF payload_monitor_id IS NOT NULL THEN
    PERFORM pg_notify(
      'monitor_events',
      json_build_object(
        'table', TG_TABLE_NAME,
        'action', TG_OP,
        'monitor_id', payload_monitor_id,
        'user_id', payload_user_id
      )::text
    );
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER llm_conversations_notify_trigger
  AFTER INSERT OR UPDATE ON llm_conversations
  FOR EACH ROW EXECUTE FUNCTION monitor_events_notify();
```

This extends the existing trigger function to handle `llm_conversations` and emits to the `monitor_events` channel with the same format (`table`, `action`, `monitor_id`, `user_id`).

### Phase 2: Update handlers and templates

1. **Update `GetLLMConversationBySourceID`** or add a new query to fetch conversations by check ID

2. **Create helper function in template package** (`internal/monitor/check_view.templ`) to map tool names to display text:
   ```go
   func toolDisplayText(call models.LLMFunctionCall) string {
       switch call.Name {
       case "browser_navigate":
           // Parse URL from arguments and display host
       case "browser_click":
           return "Clicking on a link"
       case "browser_wait":
           return "Waiting for the page to load"
       case "search_request":
           // Parse query from arguments
       default:
           return fmt.Sprintf("Running %s", call.Name)
       }
   }
   ```

3. **Update SSE handlers** in `internal/monitor/handlers.go`:
   - Handle `llm_conversations` table events from the existing `monitor_events` channel
   - Fetch and push updated conversation data when these events arrive

### Phase 3: Update UI templates

1. **Update `check_view.templ`**:
   - Remove separate `Events` field from `CheckViewData`
   - Extract timeline events from `Conversation.Messages`
   - Use the `toolDisplayText()` helper function for display text
   
2. **Update `monitor_view.templ`**:
   - Update `checkInProgressTimelineItem` to work with conversation messages
   - Use the `toolDisplayText()` helper function for display text

3. **Update `monitor_draft.templ`**:
   - Same changes as monitor_view.templ

### Phase 4: Remove check_events infrastructure

1. **Remove from LLM package**:
   - Delete `CheckEvent` struct from `check_workflow.go`
   - Delete `EventsChan` type
   - Remove channel parameter from `NewCheckWorkflow`
   - Remove `checkEvent` function from tool definitions in `tools.go`
   - Remove event channel send in `checker.go` (line 236-238)

2. **Remove from monitor package**:
   - Delete `CreateMonitorCheckEvent` function from `monitor_check_event.go`
   - Remove goroutine that consumes events in `monitor_check.go` (lines 146-155)
   - Remove `ListMonitorCheckEvents` calls from handlers
   - Delete `DeleteMonitorCheckEventsForMonitor` usage

3. **Remove from models**:
   - Delete queries from `monitors.sql`: `ListMonitorCheckEvents`, `CreateMonitorCheckEvent`, `DeleteMonitorCheckEventsForMonitor`
   - Delete `MonitorCheckEvent` struct and related types from `monitor_types.go`
   - Delete `MonitorCheckEventKind` enum

4. **Create migration to drop table**:
   ```sql
   -- up
   DROP TRIGGER IF EXISTS monitor_check_events_notify_trigger ON monitor_check_events;
   DROP FUNCTION IF EXISTS monitor_check_events_notify();
   DROP TABLE IF EXISTS monitor_check_events;
   DROP TYPE IF EXISTS monitor_check_event_kind;
   
   -- down
   -- Recreate table, enum, trigger (copy from original migration)
   ```

5. **Run `mise run sqlc-generate`** to regenerate models

### Phase 5: Update TODO.md

Remove the completed item:
```
- [ ] Refactor: could check_events and llm_conversation concepts be rolled into one?
```

## Files to Modify

| File | Changes |
|------|---------|
| `internal/db/migrations/XXXXXX_llm_conversations_notify.up.sql` | New: Add pg_notify trigger |
| `internal/db/migrations/XXXXXX_drop_check_events.up.sql` | New: Drop check_events table |
| `internal/models/monitors.sql` | Remove check_events queries |
| `internal/models/monitor_types.go` | Remove check event types |
| `internal/models/llm_conversations.go` | Add helper to extract tool calls |
| `internal/llm/check_workflow.go` | Remove EventsChan |
| `internal/llm/checker.go` | Remove event channel usage |
| `internal/llm/tools.go` | Remove `checkEvent` from tool definitions |
| `internal/monitor/monitor_check_event.go` | Delete file |
| `internal/monitor/monitor_check.go` | Remove event goroutine |
| `internal/monitor/handlers.go` | Handle `llm_conversations` events, update data fetching |
| `internal/monitor/check_view.templ` | Update timeline to use conversation, add `toolDisplayText()` helper |
| `internal/monitor/monitor_view.templ` | Update in-progress events display, use `toolDisplayText()` helper |
| `internal/monitor/monitor_draft.templ` | Update preview events display |

## Testing

1. Verify real-time timeline updates work during check execution
2. Verify completed check view shows timeline correctly
3. Verify draft monitor preview shows events correctly
4. Verify monitor deletion cascades correctly (llm_conversations should already cascade via user_id FK, but verify check-related cleanup)

## Rollback

If issues arise, the down migration recreates the check_events table. However, data from new checks would only exist in llm_conversations, so the timeline would be empty for those checks until re-checked.
