---
name: update-dependencies
description: Update dependency versions in the untils repository. Use when asked to refresh Go modules, Bun packages, vendored Datastar, or other pinned dependency versions in CI/tooling configs.
---

# Update Dependencies

Update dependencies with this workflow, then validate behavior before handing off.

## Scope

Check and update these dependency surfaces:

- Go modules: `go.mod`, `go.sum`
- Bun packages: `package.json`, `bun.lock`
- Vendored Datastar JS: `assets/js/vendor/datastar.js`
- Tooling/runtime pins: `mise.toml`
- CI and container pins: `.github/workflows/ci.yml`, `.github/actions/setup/action.yml`, `docker-compose.yml`

## Workflow

1. Capture a baseline.

- Run `jj status`.
- Record current key versions:
  - `go list -m github.com/starfederation/datastar-go`
  - `bun outdated || true`
  - `head -n 1 assets/js/vendor/datastar.js`

2. Update Go dependencies.

- Prefer patch-only updates first for safer upgrades:
  - `go get -u=patch ./...`
- If broader updates are requested, run:
  - `go get -u ./...`
- Run `go mod tidy`.
- Review remaining available Go updates with `go list -m -u all`.

3. Update Bun dependencies.

- Run `bun update --latest`.
- Run `bun install` to refresh lockfile integrity.
- Re-check with `bun outdated || true`.

4. Update vendored Datastar.

- Run `.agents/skills/update-dependencies/scripts/update_datastar_vendor.sh`.
- If a specific release is needed, pass it explicitly:
  - `.agents/skills/update-dependencies/scripts/update_datastar_vendor.sh v1.0.0-RC.8`

5. Update other useful pinned dependencies.

- Check pinned versions in config files:
  - `rg -n "golangci|image:|uses: .*@v|go =|bun =" mise.toml .github docker-compose.yml`
- Update obvious pins when appropriate, especially:
  - `golangci-lint` version in `mise.toml` and `.github/workflows/ci.yml`
  - GitHub Action major/minor tags in workflow/action files
  - Docker image tags in `docker-compose.yml` and CI services
  - `go` toolchain version in `mise.toml` and `go.mod` (only when requested or when coordinated across CI/dev)

6. Validate.

- Run `mise run lint`.
- Run `mise run test:unit`.
- Run `mise run test:integration`.
- Run `bun run build-css` and `bun run build-js`.

7. Summarize for handoff.

- List every updated file and major version movement.
- Call out any intentionally skipped updates and why.
- Include follow-up work if an update required code changes.

## Safety Rules

- Keep dependency-only changes separate from feature work.
- Prefer one dependency family at a time when debugging regressions.
- If a major upgrade introduces breakage, pin back to last known good and report the blocker.
