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

You can view the database schema in ./internal/models/schema.sql - if you modify
one of the queries, get SQLc to regenerate it with the corresponding mise task.

To connect to the database, use `psql postgresql://root:root@localhost:54324/untils_dev`

I don't like to yell at the database, lowercase the SQL you write.

## Development server

The development server will be running and uses hot reload as you make your
changes. You can view its logs at ./dev.log - however just tail the last few lines
as this file could be very big.

When the development server is running, Templ files will be generated when modified.

You may use the Chrome MCP server to access the website at address
http://untils.localhost:7331/app and log in by clicking the dev mode button.
