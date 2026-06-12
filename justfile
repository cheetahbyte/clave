set dotenv-load := true
set shell := ["bash", "-ceu"]

backend_dir := "backend"
frontend_dir := "website"

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

migrate:
    @echo "Running backend migrations..."
    goose up

reset-be:
    @echo "Rolling back all backend migrations..."
    goose reset

seed-be:
    psql "$DATABASE_URL" -f database/seed-backend.sql

gen-be:
    cd ./backend && sqlc generate

test:
    @just test-be

test-be:
    cd backend && go test ./...
