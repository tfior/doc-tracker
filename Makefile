.PHONY: dev migrate seed backend frontend

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

migrate:
	docker compose run --rm backend migrate -path /app/db/migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@db:$(DB_PORT)/$(DB_NAME)?sslmode=disable" up

seed:
	cd backend && go run ./cmd/seed
