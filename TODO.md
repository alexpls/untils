## To do

- [ ] Feature: Allow switching API providers between x.ai and OpenAI on startup
- [ ] Feature: User signup
- [ ] Feature: Monitors list page pagination
- [ ] Improvement: Feed back more of the different checks to the LLM, to help give a better answer
- [ ] Improvement: Request ID on all logs (context logger?)
- [ ] Improvement: Better use of context. Should pass it all the way down and rely less on closer functions during app startup/shutdown
- [ ] Improvement: Pass the user's timezone as context to all prompts
- [ ] Improvement: Provide additional instructions to a monitor's check (do something with the ones we're already saving from LLM)
- [ ] Refactor: Put river into its own db schema?
- [ ] Feature: Checks should be able to have multiple results
- [ ] Refactor: Forms should have some helpers extracted
- [ ] Improvement: Pushover form should show a spinner while we're validating the token
- [ ] Improvement: Triage workflow should document steps it took to get to a satisfactory answer so future workflows can do them too - possibly able to skip making new searches this way and just re-request existing URLs?
- [ ] Fix: When restarting a check (e.g. after server restart), make sure to clear old check events for it
- [ ] Fix: Need an anchored positioning tooltip polyfill for Firefox
- [ ] Refactor: Cleanup unused `monitor.expert` column
- [ ] Feature: "create new monitor" quicklink on dashboard
- [ ] Feature: handle future dates better. At the moment with the 'past tense' date requirement, release dates are referred to in the past tense, even though they haven't happened yet.

## Done

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
