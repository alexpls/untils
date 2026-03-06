## To do

### Monitor results

- [ ] Feature: Merge multiple per-check result notifications into one grouped notification payload.
- [ ] Improvement: add instructions for checking prices for things, should include things like finding the cheapest price and reporting on it. Not including multiple store names.
- [ ] Feature: display the rich fields that are being captured outside of just the headline & subtitle

### Monitor previews

- [ ] Feedback: it's unclear how often email notifications will be sent. People don't wanna get spammed so this should be much more obvious.
- [ ] Feedback: observation: non-technical people don't care so much about "how often it checks". They just care about "how often it notifies".
- [ ] Feedback: it's not obvious at all that there are multiple ways to receive notifications.
- [ ] Improvement: make it clearer what the purpose of the preview is. I've had feedback that it's confusing that it's showing something that's already happened, which isn't something that the user wants to be notified about _now_.

### AI SDKs

- [ ] Feature: Allow switching API providers between x.ai and OpenAI on startup
- [ ] Improvement: 25% of the produced binary is OpenAI's bloated SDK. I only use one endpoint, could move away from the SDK and call it directly with HTTP?

### Misc

- [ ] Feature: User signup
- [ ] Improvement: Better use of go context. Should pass it all the way down and rely less on closer functions during app startup/shutdown
- [ ] Refactor: Forms should have some helpers extracted
- [ ] Improvement: Pushover form should show a spinner while we're validating the token
- [ ] Fix: Need an anchored positioning tooltip polyfill for Firefox
- [ ] Improvement: should be able to checkpoint progress in conversation with LLM and resume it on server restart
- [ ] Refactor: make new tool creation less of a trek through various parts of the codebase. So concretely speaking, try to move as much as possible into the llm/tools package definitions.
- [ ] Fix: should be able to visit check pages for incomplete checks - but right now that errors
- [ ] Improvement: The click tool should emit the URL of the new page it landed on as a navigation event so it shows up on the UI
  - [ ] Refactor: "llm_conversations" should probably be renamed to "monitor_events", especially if it's gonna hold more than llm responses in it.
- [ ] Fix: "check now" when clicked should put the check in some kinda "queued" state, so the user has immediate feedback that their action had an effect, even if a worker may not pick it up for a while.
- [ ] Improvement: Refusal/check failures should be more obvious. At the moment say if a site returns a net::ERR_HTTP_RESPONSE_CODE_FAILURE the result will be 'no results found' when it should instead be an error

## Done

- [x] Feature: Result corrections
  - [x] Feature: hide results that were bad, and feed them back to LLM
  - [x] Feedback: it's not clear that the "feedback" option goes to the LLM to influence future results. People were trying to give feedback about the application itself in that field.
  - [x] Feature: change this to "corrections" instead of "feedback", and let the actual response text be corrected. It's annoying to prompt your way to the response shape you want when you could instead just correct it yourself and have the LLM learn from it.
- [x] Improvement: Pass the user's timezone as context to all prompts
- [x] Fix: (requires https://github.com/starfederation/datastar/issues/900 first) event sse should not deliver a message when subscribed to right after a visit to a page. But they should send a message when subscribed to on reentry to a page (e.g. switching back to the tab). On Dashboard page, only the first subscription should use view transitions. (not actually done but I don't remember what I wanted to do here)
- [x] Feature: Change password
- [x] Have some docs that the LLM can consult on demand for how to format things, and a general index that it can use to pull up docs. e.g. a doc about formatting fields for podcast episodes could be that the headline should be "#{{Episode number}} - {{Episode title}}" and subtitle should be "Released {{Release date}}".
- [x] Feedback: it's unclear that the monitor isn't already ready to go when it's being previewed. When the first "preview" result comes in there's an assumption that the thing is already active.
- [x] Feedback: "start monitoring now" button should probably just say "Activate".
- [x] Feature: "fire and forget" way to set up monitors, if you don't wanna sit around waiting to confirm that the first check looks good
- [x] Improvement: Triage workflow should document steps it took to get to a satisfactory answer so future workflows can do them too - possibly able to skip making new searches this way and just re-request existing URLs?
- [x] Feature: Checks should be able to produce multiple results
- [x] Feature: handle future dates better. At the moment with the 'past tense' date requirement, release dates are referred to in the past tense, even though they haven't happened yet.
- [x] Refactor: Abstract out calls to the underlying LLM provider so we can swap 'em out on the fly
- [x] Fix: if running a river job right away, make sure it's not 'scheduled', but rather 'available' - see https://riverqueue.com/docs/scheduled-jobs
- [x] Fix: updating the check schedule should modify the currently scheduled checks
- [x] Refactor: can I get rid of skipped checks? It's annoying and doesn't make sense on the UI.
- [x] Fix: When restarting a check (e.g. after server restart), make sure to clear old check events for it
- [x] Fix: sometimes a draft will get stuck on 'previewing' status and will need a page refresh to actually show the "Activate" screen
- [x] Fix: check timeline event timestamps aren't right aligned. they should be, and should have a small margin on their left for good measure.
- [x] Refactor: could check_events and llm_conversation concepts be rolled into one? They have some crossover currently
- [x] Improvement: page titles
- [x] Fix: when timeago is "5 hours 59 minutes in the future" it will return "in 5 hours", when it should really round to the nearest hour and return "in 6 hours"
- [x] Improvement: Request ID on all logs (context logger?)
- [x] Fix: why did a panic happen during job checks where previewing tried to transition to validating?
- [x] Improvement: The search tool should not be able to be called with the same URL multiple times in a row
- [x] Fix: don't allow changing the subject while monitor is previewing. That's an invalid status transition.
- [x] Fix: Did i break feedback?
- [x] Fix: draft monitors should show up on monitor page as drafts
- [x] Fix: all pages responding with SSE events should send an event as soon as a connection occurs, this is to support people refreshing page or navigating back to the page.
- [x] Fix: hitting "enter" when correcting a rejection reason resets the input
- [x] Should stop jobs being considered "stuck" prematurely: https://github.com/riverqueue/river/issues/1125
- [x] Fix: Refreshing the monitor draft page while a monitor is in 'previewing' status returns 500 (page refresh triggers a POST that attempts invalid transition from 'previewing' to 'validating')
- [x] Fix: why does it take so long to browse sites? Doesn't seem right...
- [x] Feature: Monitors list page pagination
- [x] Refactor: in paths, instead of "id" for the monitor id it should always be "monitor_id". In other words, IDs in paths should be identifiable.
- [x] Feature: "create new monitor" quicklink on dashboard
- [x] Improvement: Feed back more of the different checks to the LLM, to help give a better answer
- [x] Fix: bump validator timeout, it's at 1 minute at the moment
- [x] Refactor: Cleanup unused `monitor.expert` column, also `monitor.instructions`.
- [x] Refactor: Breadcrumbs and page titles should be coupled somehow so they don't need to be redeclared everywhere
- [x] Refactor: Integration should not be "Active" but "Configured". This would disambiguate it from the notifiers set up on monitors, which are also "active" when a user has enabled them.
- [x] Improvement: Check should return favicon location of the sites it's visiting
- [x] Improvement: UI auto-updates when checks start
- [x] Improvement: Monitor draft page refresh should not retrigger POST, but instead GET
- [x] Refactor: make monitor notifiers more generic (e.g. for email later)
- [x] Feature: User settings (timezone, notification integrations config)
- [x] Improvement: Figure out html template formatting
- [x] Feature: URL for result is cited and clickable
- [x] Feature: Delete monitor
- [x] Refactor: Extract monitor optimistic locks to helper method
- [x] Refactor: Pushover client should be consolidated a bit
- [x] Refactor: Get rid of transaction boilerplate
- [x] Refactor: Get rid of useless updated_at
- [x] Feature: Guided monitor creation
- [x] Improvement: Datetimes are formatted more nicely
- [x] Bug: 'cannot scan null into string' when ListMonitorsWithChecks has a monitor with no latest check result
- [x] Bug: Test that if I stop a check worker halfway through it recovers eventually
- [x] Improvement: Input validation (e.g. for user timezone)
- [x] Use `different_to_previous` param from JSON response to set the flag in the DB, rather than checking for string equality.
- [x] Log SQL queries
- [x] River logger should use slog logger
- [x] CSRF
