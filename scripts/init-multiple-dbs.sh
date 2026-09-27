#!/bin/bash

set -e

databases=(
  "order_db"
  "inventory_db"
  "payment_db"
  "shipping_db"
  "notification_db"
)

for db in "${databases[@]}"; do
  echo "Creating database: $db"
  psql -v ON_ERROR_STOP=1 \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" \
    -c "CREATE DATABASE $db;"
done