## To do

- [ ] Feature: Allow switching API providers between x.ai and OpenAI on startup
- [ ] Feature: User signup
- [ ] Improvement: Request ID on all logs (context logger?)
- [ ] Improvement: Better use of context. Should pass it all the way down and rely less on closer functions during app startup/shutdown
- [ ] Improvement: Pass the user's timezone as context to all prompts
- [ ] Refactor: Put river into its own db schema?
- [ ] Feature: Checks should be able to have multiple results
- [ ] Refactor: Forms should have some helpers extracted
- [ ] Improvement: Pushover form should show a spinner while we're validating the token
- [ ] Improvement: Triage workflow should document steps it took to get to a satisfactory answer so future workflows can do them too - possibly able to skip making new searches this way and just re-request existing URLs?
- [ ] Improvement: The search tool should not be able to be called with the same URL multiple times in a row
- [ ] Improvement: The click tool should emit the URL of the new page it landed on as a navigation event so it shows up on the UI
- [ ] Fix: When restarting a check (e.g. after server restart), make sure to clear old check events for it
- [ ] Fix: Need an anchored positioning tooltip polyfill for Firefox
- [ ] Feature: handle future dates better. At the moment with the 'past tense' date requirement, release dates are referred to in the past tense, even though they haven't happened yet.
- [ ] Feature: "fire and forget" way to set up monitors, if you don't wanna sit around waiting to confirm that the first check looks good
- [ ] Fix: don't allow changing the subject while monitor is previewing. That's an invalid status transition.
- [ ] Fix: Did i break feedback?
- [ ] Fix: when timeago is "5 hours 59 minutes in the future" it will return "in 5 hours", when it should really round to the nearest hour and return "in 6 hours"
- [ ] Fix: draft monitors should show up on monitor page as drafts
- [ ] Improvement: should be able to checkpoint progress in conversation with LLM and resume it on server restart
- [ ] Fix: why did a panic happen during job checks where previewing tried to transition to validating?

## Watching

## Done

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
