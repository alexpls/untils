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

## Database

- You can view the database schema in ./internal/models/schema.sql - never modify this directly,
  use the mise tasks instead.
- To connect to the database, use `psql postgresql://root:root@localhost:54324/untils_dev`.
- I don't like to yell at the database, lowercase any SQL you write.

## Development server

The development server will be running and uses hot reload as you make your
changes. You can view its logs at ./dev.log - however just tail the last few lines
as this file could be very big.

When the development server is running, Templ files will be generated when modified.

You may use the Chrome MCP server to access the website at address
http://untils.localhost:7331/app and log in by clicking the dev mode button.

## Writing UI copy

- Prefer impersonal, non-agentive phrasing for user-facing text.
- Avoid first-person references like "we", "our", or "us".
- Use direct product behavior language, for example: "notifications will be sent when it changes."
