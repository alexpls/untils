#!/bin/bash
set -euo pipefail

# Backport staging database to local development environment
#
# Prerequisites:
#   - SSH access to "fed" server
#   - Local PostgreSQL running for the active workspace (via docker compose)
#
# Visudo entry for passwordless docker exec on fed:
#   alex ALL=(ALL) NOPASSWD: /usr/bin/docker exec untils_go-db-1 *

REMOTE_HOST="fed"
REMOTE_CONTAINER="untils_go-db-1"
REMOTE_DB_USER="root"
DB_NAME="untils_dev"
DUMP_FILE="./tmp/untils_staging_dump_$$.sql"

echo "==> Dumping staging database from ${REMOTE_HOST}..."
ssh "${REMOTE_HOST}" "sudo docker exec ${REMOTE_CONTAINER} pg_dump -U ${REMOTE_DB_USER} -d ${DB_NAME} --clean --if-exists" > "${DUMP_FILE}"

echo "==> Recreating local database ${DB_NAME}..."
mise run db:drop && mise run db:create

echo "==> Importing dump to local database..."
psql "${PG_URL}" -f "${DUMP_FILE}"

echo "==> Running migrations..."
mise run db:migrate:up

echo "==> Done! Staging database has been imported to local ${DB_NAME}."
