-include .env
export

.PHONY: dev migrate seed create-user backend frontend

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

migrate:
	migrate -path backend/db/migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" up

seed:
	cd backend && go run ./cmd/seed

create-user:
	cd backend && go run ./cmd/create-user
