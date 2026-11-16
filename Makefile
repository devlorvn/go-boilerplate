ENV?=local
GENPROTO_OUT=/pkg/genproto
TEST_FILE=.

.PHONY: env run-gateway run-auth run-file run-all gen-proto migrate-up migrate-down migrate-auth-up migrate-auth-down migrate-file-up migrate-file-down dev-up dev-down build-all test

# Create local env files from the committed templates (does not overwrite)
env:
	@for s in auth file gateway; do [ -f .env.$$s ] || cp .env.$$s.example .env.$$s; done

# Individual Service Runners
run-gateway:
	go run cmd/gateway/main.go

run-auth:
	go run cmd/authservice/main.go

run-file:
	go run cmd/fileservice/main.go

# Run All Microservices Concurrently (Dev Mode)
run-all:
	go run cmd/authservice/main.go & go run cmd/fileservice/main.go & go run cmd/gateway/main.go

# Proto & Database Migrations
gen-proto:
	go run scripts/gen_proto/main.go

migrate-up:
	go run scripts/migrate/main.go all up

migrate-down:
	go run scripts/migrate/main.go all down

migrate-auth-up:
	go run scripts/migrate/main.go auth up

migrate-auth-down:
	go run scripts/migrate/main.go auth down

migrate-file-up:
	go run scripts/migrate/main.go file up

migrate-file-down:
	go run scripts/migrate/main.go file down

# Infrastructure Management (Docker Compose)
dev-up:
	docker compose up -d

dev-down:
	docker compose down

# Build All Binaries
build-all:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/authservice ./cmd/authservice
	go build -o bin/fileservice ./cmd/fileservice

# Run Tests
test:
	go test -v ./...
