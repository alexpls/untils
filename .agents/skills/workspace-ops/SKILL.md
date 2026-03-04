---
name: workspace-ops
description: Manage jj workspaces in untils, including create/bootstrap and deterministic teardown. Use when asked to perform workspace setup, cleanup, or other workspace operations.
---

# Workspaces

Use this workflow when asked to create and fully initialize a new `jj workspace`.

## Inputs

- Workspace name (example: `workspacetest`)
- Optional destination path. Default: sibling directory `../untils-<workspace-name>`

## Workflow

Use the deterministic bootstrap script from repo root:

- `.agents/skills/workspace-ops/scripts/create_workspace.sh <workspace-name> [destination-path]`

What it does:

- Creates the workspace via `jj workspace add`.
- Runs `bun install` and `mise trust` in the new workspace.
- Starts workspace-scoped services via `mise run dev:up`.
- Initializes databases with `mise run db:reset`.
- Builds frontend assets (`bun run build-css`, `bun run build-js`).
- Verifies seed user presence, runs `mise run test:unit`, and prints `mise run dev:info`.

Rules:

- Prefer this script instead of ad-hoc manual setup commands.
- If a custom destination path is needed, pass it as the second argument.
- Start the dev server separately when requested: `mise run dev`.

## Deleting a workspace

Use the deterministic teardown script from repo root:

- `.agents/skills/workspace-ops/scripts/delete_workspace.sh <workspace-name>`

What it does:

- Stops workspace-scoped docker services (`mise run dev:down`) when the workspace directory exists.
- Forgets the workspace in `jj`.
- Deletes the workspace directory.
- Removes the workspace docker volume (`untils_<workspace-slug>_pgdata`).

Rules:

- Never delete `default`.
- Prefer this script instead of ad-hoc manual commands.

## Expected result

- Workspace has isolated docker project/ports based on workspace name.
- `untils_dev` and `untils_test` are ready.
- Seed user can sign in:
  - email: `alexpls@fastmail.com`
  - password: `abc123`
- Unit tests pass and the app can run independently of other workspaces.
