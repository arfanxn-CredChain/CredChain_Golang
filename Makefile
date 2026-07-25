# CredChain — orchestrator + backend
# Run everything from here. Prod = full Docker. Local = hybrid (infra in Docker,
# Go via air + React via vite on host).

ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
    include $(ENV_FILE)
    export
endif

.PHONY: help up down fresh logs backup restore dev-up dev-fresh dev \
	test lint get-google-id-token credential-extraction-benchmark wait-golang

help:
	@echo "CredChain — make targets"
	@echo ""
	@echo "Stack (Docker / prod):"
	@echo "  up        Bring up the whole stack in Docker (contracts, migrate, super-admin, all services)"
	@echo "  down      Stop all containers"
	@echo "  fresh     Wipe volumes + uploads, then up"
	@echo "  logs      Follow backend/infra logs"
	@echo "  backup    Dump Postgres + Mongo + uploads      (BACKUP=<ts>)"
	@echo "  restore   Restore from a backup                 (BACKUP=<ts>)"
	@echo ""
	@echo "Local (hybrid):"
	@echo "  dev-up    Infra + Anvil + Python in Docker, deploy contracts, migrate, seed"
	@echo "  dev-fresh Wipe local infra data + uploads, then dev-up (clean chain + reseed)"
	@echo "  dev       Run Go backend hot-reload (air). Then run 'make dev' in CredChain_React."
	@echo ""
	@echo "Backend:"
	@echo "  test      go test ./..."
	@echo "  lint      go vet + gofmt"
	@echo "  get-google-id-token          Obtain Google ID token (Postman)"
	@echo "  credential-extraction-benchmark   Run extraction benchmark"

# ---------------------------------------------------------------- full stack (Docker)

up:
	-docker network create credchain
	docker compose up -d anvil postgres mongo
	python3 scripts/setup-contracts.py
	docker compose up -d --build golang
	@$(MAKE) wait-golang
	docker compose run --rm golang ./server migrate up
	-docker compose run --rm golang ./server migrate-mongo up
	-docker compose exec golang ./server init-super-admin
	cd ../CredChain_React && docker compose --env-file .env.docker up -d --build
	docker compose up -d nginx
	cd ../CredChain_Python && docker compose up -d --build
	@echo "up: all services started"

down:
	-cd ../CredChain_React && docker compose down
	-cd ../CredChain_Python && docker compose down
	docker compose down

fresh:
	@$(MAKE) down
	rm -rf docker/postgres/data/* docker/mongo/data/* docker/anvil/data/* uploads/*
	@$(MAKE) up

logs:
	docker compose logs -f

# ---------------------------------------------------------------- local (hybrid)

dev-up:
	-docker network create credchain
	docker compose up -d anvil postgres mongo
	cd ../CredChain_Python && docker compose up -d --build
	ENV_FILE=.env python3 scripts/setup-contracts.py
	go run main.go migrate up --env .env
	-go run main.go migrate-mongo up --env .env
	go run main.go seed --env .env
	go run main.go seed-chain --env .env
	@echo "dev-up ready. Run 'make dev' here, and 'make dev' in CredChain_React."

dev-fresh:
	-cd ../CredChain_Python && docker compose down
	docker compose down
	rm -rf docker/postgres/data/* docker/mongo/data/* docker/anvil/data/* uploads/*
	@$(MAKE) dev-up

dev:
	@command -v air >/dev/null 2>&1 || { echo "error: air not installed. Run: go install github.com/air-verse/air@latest"; exit 1; }
	air -c .air.toml

# ---------------------------------------------------------------- backend tasks

test:
	go test ./...

lint:
	go vet ./... && gofmt -l .

get-google-id-token:
	go run main.go get-google-id-token --env $(ENV_FILE)

CREDENTIAL_EXTRACTION_BENCH_COUNT ?= 3
CREDENTIAL_EXTRACTION_BENCH_DIRECTORY ?= benchmarks/credential-extraction
CREDENTIAL_EXTRACTION_BENCH_CSV ?=
CREDENTIAL_EXTRACTION_BENCH_PROFILE ?=

credential-extraction-benchmark:
	mkdir -p $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)
	@FLAGS="" && \
	if [ -n "$(CREDENTIAL_EXTRACTION_BENCH_CSV)" ]; then FLAGS="$$FLAGS --output $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/results.csv"; fi && \
	if [ -n "$(CREDENTIAL_EXTRACTION_BENCH_PROFILE)" ]; then FLAGS="$$FLAGS --cpuprofile $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/cpu.prof --memprofile $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/mem.prof --trace $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/trace.out"; fi && \
	go run main.go credential-extraction-benchmark \
		--count $(CREDENTIAL_EXTRACTION_BENCH_COUNT) \
		$$FLAGS \
		--env $(ENV_FILE)

# ---------------------------------------------------------------- backup / restore
# Creds read from each container's own env (.env.docker), so no Makefile var juggling.

BACKUP ?= $(shell date +%Y%m%d_%H%M%S)

backup:
	@mkdir -p docker/backups
	docker compose exec -T postgres sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -Fc -U $$POSTGRES_USER $$POSTGRES_DB' > docker/backups/pg_$(BACKUP).dump
	docker compose exec -T mongo sh -c 'mongodump --archive -u $$MONGO_INITDB_ROOT_USERNAME -p $$MONGO_INITDB_ROOT_PASSWORD --authenticationDatabase admin' > docker/backups/mongo_$(BACKUP).archive
	tar czf docker/backups/uploads_$(BACKUP).tar.gz uploads
	@echo "backup $(BACKUP)"

restore:
	@test -n "$(BACKUP)" || { echo "error: set BACKUP=<timestamp>, e.g. make restore BACKUP=20260725_120000"; exit 1; }
	docker compose exec -T postgres sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_restore -U $$POSTGRES_USER -d $$POSTGRES_DB --clean' < docker/backups/pg_$(BACKUP).dump
	docker compose exec -T mongo sh -c 'mongorestore --archive --drop -u $$MONGO_INITDB_ROOT_USERNAME -p $$MONGO_INITDB_ROOT_PASSWORD --authenticationDatabase admin' < docker/backups/mongo_$(BACKUP).archive
	tar xzf docker/backups/uploads_$(BACKUP).tar.gz
	@echo "restore $(BACKUP) done"

# ---------------------------------------------------------------- helpers

wait-golang:
	@echo "waiting for golang to be healthy..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker compose ps golang | grep -q "(healthy)"; then echo "golang is healthy"; exit 0; fi; \
		echo "waiting... ($$i/10)"; sleep 3; \
	done; \
	echo "error: golang did not become healthy in time"; exit 1
