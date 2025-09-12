#!/bin/sh

# Docker entrypoint script for expense management application
set -e

echo "Starting Expense Management..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done

echo "PostgreSQL is ready!"

# Run database migrations
echo "Running database migrations..."
./main migrate up

# Run database seeding
echo "Seeding database with initial data..."
./main seed

# Start the application
echo "Starting the application..."
exec ./main server
