# Untils

This file is for coding agents working with this project.

---

"Untils" lets users set up monitors for things they care about and get notified
when they change.

Backend:

- Go http server
- Templ for templating
- PostgreSQL database
- SQLc for SQL query generation

Frontend:

- Tailwind 4 CSS
- Daisy UI
- Datastar (ref: https://data-star.dev/reference/attributes)

## Dev tasks

Run `mise tasks` to view the standard development tasks that you can run. These include
things related to code generation, db migrations, testing, linting, etc.

## jj workspaces

- When working from a newly created `jj workspace`, run `bun install` in that workspace directory before running dev tasks.
- This ensures JS/CSS tooling dependencies are available for that workspace if they have not already been installed there.
- Workspace runtime settings are derived from the active workspace name via `./scripts/workspace-env.sh`.
- Use `mise run dev:info` to print the current workspace's app/db/compose settings.
- Use `mise run dev:up` and `mise run dev:down` to start/stop the workspace's docker services.
- A dedicated skill exists at `.agents/skills/workspace-ops/SKILL.md` for workspace setup and teardown operations.

## Database

- You can view the database schema in ./internal/models/schema.sql - never modify this directly,
  use the mise tasks instead.
- The default workspace database is `postgresql://root:root@localhost:54324/untils_dev`.
- For non-default workspaces, run `mise run dev:info` and use the printed DB URL.
- I don't like to yell at the database, lowercase any SQL you write.

## Development server

The development server will be running and uses hot reload as you make your
changes. You can view its logs at ./dev.log - however just tail the last few lines
as this file could be very big.

When the development server is running, Templ files will be generated when modified.

For the default workspace, you may use the Chrome MCP server to access
http://untils.localhost:7331/app and log in by clicking the dev mode button.
For non-default workspaces, use `mise run dev:info` and open the printed localhost app URL.

## Writing UI copy

- Prefer impersonal, non-agentive phrasing for user-facing text.
- Avoid first-person references like "we", "our", or "us".
- Use direct product behavior language, for example: "notifications will be sent when it changes."

## Writing code

- Fail fast if something is not in an expected state. e.g. if an expected app config value is missing, just panic.
