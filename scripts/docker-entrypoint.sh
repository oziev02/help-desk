#!/bin/sh
set -e

echo "waiting for database..."
# migrate retries connection; give postgres a moment after healthcheck
sleep 1

echo "applying migrations..."
migrate -path /migrations -database "$DATABASE_URL" up

echo "starting server..."
exec /app/help-desk
