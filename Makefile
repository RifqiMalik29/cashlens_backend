.PHONY: run build test migrate-up migrate-down migrate-create

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

migrate-up:
	goose -dir migrations postgres "${DATABASE_URL}" up

migrate-down:
	goose -dir migrations postgres "${DATABASE_URL}" down

migrate-create:
	goose -dir migrations create $(name) sql
