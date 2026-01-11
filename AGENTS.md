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

## Database schema

You can view the database schema in ./internal/db/models/schema.sql - if you modify
one of the queries, you can get SQLc to regenerate it with `mise run sqlc-generate`.

## Development server

The development server will be running and uses hot reload as you make your
changes. You can view its logs at ./dev.log - however just tail the last few lines
as this file could be very big.

When the development server is running, Templ files will be generated when modified.

You may use the Chrome MCP server to access the website at address
http://untils.localhost:7331/app and log in by clicking the dev mode button.
