<div align="center">
  <h1>
    <a href="https://untils.com"><img src="public/images/logo.png" alt="untils logo" width="36" height="36" align="center"></a>
    untils
  </h1>
  <p>
    AI monitoring of pages, products, and signals that should not be missed.
  </p>
  <p>
    <a href="https://untils.com">Website</a> ·
    <a href="https://untils.com/docs">Docs</a>
  </p>
</div>

---

Still under active development. See the [home page](https://untils.com)
for more information and join the waitlist for updates about its release.

If you want earlier access, you can self-host untils. But you're reading
this README so you probably already know that :)
The [self-hosting quickstart](https://untils.com/docs/self-hosting/quickstart)
doc is a good place to learn more.

![List of monitors in untils](public/images/screenshots/full/untils_monitor_list.jpg)

---

## Stack

Backend: [Go](https://go.dev/), [PostgreSQL](https://www.postgresql.org/), [sqlc](https://sqlc.dev/), [River](https://riverqueue.com/)

Frontend: [templ](https://templ.guide/), [Tailwind CSS 4](https://tailwindcss.com/), [DaisyUI](https://daisyui.com/), [Datastar](https://data-star.dev/).

## Development setup

### Prerequisites

Install:

- [mise](https://mise.jdx.dev/) for Go and Bun versions/tasks
- Docker with `docker compose`
- PostgreSQL client tools (`psql`, `pg_dump`)
- Git or Jujutsu (`jj`), depending on workflow

Optional but needed for fully exercising monitor checks:

- Brave Search API key
- OpenAI-compatible LLM API key, such as OpenAI or x.ai
- Pushover app key and/or SMTP settings for notification delivery

### 1. Install language/tool versions

```sh
mise install
bun install
mise run dev:deps # installs go dev tooling
```

### 2. Create `.env.dev`

`.env.dev` is intentionally ignored by Git. Create one for local development:

```sh
cat > .env.dev <<'EOF'
ENV=dev
APP_MODE=selfhosted
ADMIN_EMAIL=alex@example.com

# Optional for real monitor checks. Blank values are enough to start the app,
# but checks that call these providers will fail until real keys are configured.
OPENAI_API_KEY=
OPENAI_MODEL=grok-4-1-fast-reasoning
BRAVE_KEY=

SMTP_HOST=127.0.0.1
SMTP_PORT=1025
SMTP_FROM=notifications@untils.local
EOF

mise run dev:env:sync
```

git worktrees / jj workspaces are supported out of the box, so multiple instances
of the application can be developed at the same time.

`dev:env:sync` adds workspace-specific ports and database URLs to `.env.dev`. The default workspace uses:

- app: `http://localhost:4200/app`
- templ proxy: `http://localhost:7331/app`
- database: `postgresql://root:root@localhost:54324/untils_dev`
- Mailpit: `http://localhost:8025`
- River UI: `http://localhost:7332`

Run `mise run dev:info` at any time to print the current workspace settings.

### 3. Start services and prepare the database

```sh
mise run dev:up
mise run db:reset
```

This starts PostgreSQL, Mailpit, and River UI, then creates, migrates, seeds, and regenerates sqlc output for the development and test databases.

The seed user is:

- email: `alex@example.com`
- password: `abc123`

In `ENV=dev`, the sign-in page also shows a development sign-in button.

### 4. Run the development server

```sh
mise run dev
```

This runs the templ/app watcher plus CSS and JS watchers. Open the templ proxy URL from `mise run dev:info` for hot reload.

Development logs are written to [`dev.log`](dev.log) when the watcher is running.

## Common tasks

List all tasks:

```sh
mise tasks
```

Frequently used tasks:

```sh
mise run dev              # run app, css, and js watchers
mise run dev:up           # start docker services
mise run format           # auto-format files
mise run lint             # lint
mise run test:unit        # run unit tests
mise run test:integration # run integration tests (require API keys)
```

## License

See [`LICENSE.md`](LICENSE.md).
