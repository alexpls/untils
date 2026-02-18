#!/bin/bash
set -euo pipefail

# Backport staging database to local development environment
#
# Prerequisites:
#   - SSH access to "fed" server
#   - Local PostgreSQL running on port 54324 (via docker-compose)
#
# Visudo entry for passwordless docker exec on fed:
#   alex ALL=(ALL) NOPASSWD: /usr/bin/docker exec untils_go-db-1 *

REMOTE_HOST="fed"
REMOTE_CONTAINER="untils_go-db-1"
REMOTE_DB_USER="root"
REMOTE_DB_NAME="untils_dev"

LOCAL_DB_HOST="localhost"
LOCAL_DB_PORT="54324"
LOCAL_DB_USER="root"
LOCAL_DB_PASSWORD="root"
LOCAL_DB_NAME="untils_dev"
LOCAL_DB_ADMIN_DB="postgres"

DUMP_FILE="/tmp/untils_staging_dump_$$.sql"

echo "==> Dumping staging database from ${REMOTE_HOST}..."
ssh "${REMOTE_HOST}" "sudo docker exec ${REMOTE_CONTAINER} pg_dump -U ${REMOTE_DB_USER} -d ${REMOTE_DB_NAME} --clean --if-exists" > "${DUMP_FILE}"

echo "==> Recreating local database ${LOCAL_DB_NAME}..."
PGPASSWORD="${LOCAL_DB_PASSWORD}" psql \
    -h "${LOCAL_DB_HOST}" \
    -p "${LOCAL_DB_PORT}" \
    -U "${LOCAL_DB_USER}" \
    -d "${LOCAL_DB_ADMIN_DB}" \
    -v ON_ERROR_STOP=1 \
    -c "select pg_terminate_backend(pid) from pg_stat_activity where datname = '${LOCAL_DB_NAME}' and pid <> pg_backend_pid();" \
    -c "drop database if exists ${LOCAL_DB_NAME};" \
    -c "create database ${LOCAL_DB_NAME};"

echo "==> Importing dump to local database..."
PGPASSWORD="${LOCAL_DB_PASSWORD}" psql \
    -h "${LOCAL_DB_HOST}" \
    -p "${LOCAL_DB_PORT}" \
    -U "${LOCAL_DB_USER}" \
    -d "${LOCAL_DB_NAME}" \
    -f "${DUMP_FILE}"

echo "==> Cleaning up..."
rm -f "${DUMP_FILE}"

echo "==> Done! Staging database has been imported to local ${LOCAL_DB_NAME}."
