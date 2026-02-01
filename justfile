set dotenv-load := true
set shell := ["bash", "-ceu"]

backend_dir := "backend"
frontend_dir := "frontend"

_default:
    @just --list

help:
    @just --list

dev:
    @just dev-be & just dev-fe & wait

dev-be:
    cd {{backend_dir}} && air

dev-fe:
    cd {{frontend_dir}} && pnpm dev

migrate: migrate-be migrate-fe

migrate-be:
    @echo "Running backend migrations..."
    goose up

migrate-fe:
    @echo "Running frontend migrations (Drizzle)..."
    cd {{frontend_dir}} && pnpm drizzle-kit migrate

reset-be:
    @echo "Rolling back all backend migrations..."
    goose reset

seed-be:
    psql "$DATABASE_URL" -f database/seed-backend.sql

gen-be:
    cd ./backend && sqlc generate
