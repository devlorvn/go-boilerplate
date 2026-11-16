DB_SERVICES := auth file
CONFIG_DIR  ?= configs
SERVER      := go run ./cmd/server

.PHONY: secrets run-all migrate-up build test gen-proto dev-up dev-down

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
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./...

gen-proto:
	go run scripts/gen_proto/main.go

dev-up:
	docker compose up -d

dev-down:
	docker compose down
