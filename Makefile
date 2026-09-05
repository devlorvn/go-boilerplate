include .env
.EXPORT_ALL_VARIABLES:

ENV?=local
GENPROTO_OUT=/pkg/genproto
TEST_FILE=.

.PHONY: run gen-proto migrate-up migrate-down dev-up dev-down test

run:
	go run cmd/authservice/main.go

gen-proto:
	go run scripts/gen_proto/main.go

migrate-up:
	go run scripts/migrate/main.go up

migrate-down:
	go run scripts/migrate/main.go down

dev-up:
	docker compose up -d

dev-down:
	docker compose down

test:
	go test -v ./...
