set dotenv-load
set shell := ["bash", "-ceu"]

backend_dir := "backend"
frontend_dir := "website"
emailer_dir := "emailer"
infra_services := "postgres mailpit rabbitmq"

_default:
    @just --list

help:
    @just --list

dev:
    @just dev-be & just dev-fe & just dev-emailer & wait

dev-infra:
    docker compose -f compose.dev.yaml up -d {{ infra_services }}

dev-be:
    cd {{ backend_dir }} && air

dev-fe:
    cd {{ frontend_dir }} && pnpm dev

dev-emailer:
    cd {{ emailer_dir }} && bun dev

migrate:
    @echo "Running backend migrations..."
    goose -dir backend/migrations up

reset-be:
    @echo "Rolling back all backend migrations..."
    goose -dir backend/migrations reset

seed-be:
    psql "$DATABASE_URL" -f database/seed-backend.sql

gen-be:
    cd ./backend && sqlc generate

test:
    @just test-be

test-be:
    cd backend && go test ./...
