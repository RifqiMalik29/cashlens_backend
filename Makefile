.PHONY: run build test migrate-up migrate-down migrate-create db-setup

# Use absolute path for goose since it's not in the shell's PATH
GOOSE := $(shell go env GOPATH)/bin/goose

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

migrate-up:
	@export $$(grep -v '^#' .env | xargs) && $(GOOSE) -dir migrations postgres "$${DATABASE_URL}" up

migrate-down:
	@export $$(grep -v '^#' .env | xargs) && $(GOOSE) -dir migrations postgres "$${DATABASE_URL}" down

migrate-create:
	@$(GOOSE) -dir migrations create $(name) sql

db-setup:
	@echo "Creating database and user..."
	@psql postgres -c "CREATE USER cashlens WITH PASSWORD 'cashlens_dev';" || true
	@psql postgres -c "CREATE DATABASE cashlens OWNER cashlens;" || true
	@psql postgres -c "ALTER USER cashlens CREATEDB;" || true
	@make migrate-up
	@echo "Database setup complete!"

run-postgressql:
	brew services start postgresql@16

run-postgres:
	psql postgres