#!/bin/sh
set -e

echo "==> Running database migrations..."
./migrate
echo "==> Migrations done. Starting server..."
exec ./server
