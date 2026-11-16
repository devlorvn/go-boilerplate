DB_SERVICES := auth file
CONFIG_DIR  ?= configs
SERVER      := go run ./cmd/server
IMAGE       ?= go-boilerplate:local
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Tool versions, kept in sync with .github/workflows/ci.yml
GOLANGCI_LINT_VERSION      := v2.15.0
BUF_VERSION                := v1.75.0
PROTOC_GEN_GO_VERSION      := v1.36.11
PROTOC_GEN_GO_GRPC_VERSION := v1.6.2

.PHONY: secrets run-all migrate-up build test cover lint fmt gen-proto lint-proto breaking \
	docker-build up-app down-app dev-up dev-down tools

# Create gitignored configs/secrets/{auth,file}.yaml sharing one random JWT secret (never overwrites)
secrets:
	@secret=$$(openssl rand -hex 32); for s in auth file; do \
		f=$(CONFIG_DIR)/secrets/$$s.yaml; [ -f $$f ] && continue; \
		sed "s/change-me .*/$$secret/" $(CONFIG_DIR)/secrets/example.yaml | grep -v '^#' > $$f; echo "created $$f"; \
	done

# make run-auth | run-file | run-gateway
run-%:
	$(SERVER) gserver $* --config-dir $(CONFIG_DIR)

run-all:
	@trap 'kill 0' INT TERM; \
	$(SERVER) gserver auth & $(SERVER) gserver file & sleep 3; $(SERVER) gserver gateway & wait

migrate-up:
	@for s in $(DB_SERVICES); do $(SERVER) migrate $$s up --config-dir $(CONFIG_DIR) || exit 1; done

# make migrate-down-auth STEPS=1
migrate-down-%:
	$(SERVER) migrate $* down --steps $(or $(STEPS),1) --config-dir $(CONFIG_DIR)

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/server ./cmd/server

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

gen-proto:
	buf generate

lint-proto:
	buf lint

# Fails if proto changes break clients compared with origin/main
breaking:
	buf breaking --against '.git#branch=origin/main,subdir=proto'

# Install the dev tools above into $(go env GOPATH)/bin
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

# Whole stack in Docker: infra + migrations + auth, file, gateway
up-app:
	JWT_SECRET=$${JWT_SECRET:-$$(openssl rand -hex 32)} docker compose --profile app up -d --build --wait

down-app:
	docker compose --profile app down

dev-up:
	docker compose up -d

dev-down:
	docker compose down
