# CredChain - Makefile Commands

ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
    include $(ENV_FILE)
    export
endif

.PHONY: help check-env check-env-docker clean build serve dev migrate-up migrate-down migrate-up-mongo migrate-down-mongo init-super-admin get-google-id-token seed seed-chain \
	docker-migrate-up docker-migrate-down docker-up-build docker-up \
	docker-down docker-restart docker-logs docker-ps docker-fresh \
	docker-clean-data docker-check-backend-healthy \
	docker-backup docker-restore \
	credential-extraction-benchmark

help:
	@echo "CredChain - Available Commands:"
	@echo ""
	@echo "Usage:"
	@echo "  make serve ENV=.env.docker      # Use custom env file"
	@echo ""
	@echo "Local Development:"
	@echo "  make build              - Build the application"
	@echo "  make serve             - Start the application server"
	@echo "  make dev               - Start the application server with hot reload (requires air)"
	@echo "  make migrate-up        - Run database migrations up (local)"
	@echo "  make migrate-down      - Rollback database migrations (local)"
	@echo "  make init-super-admin  - Create super admin user (local)"
	@echo "  make get-google-id-token - Obtain Google ID token via OAuth (for Postman)"
	@echo "  make seed              - Run database seeders (populate 15 users)"
	@echo "  make seed-chain        - Register seeded user roles on-chain"
	@echo ""
	@echo "Benchmark:"
	@echo "  make credential-extraction-benchmark        - Run credential extraction benchmark"
	@echo "  Pipeline: generate PDF -> encrypt -> decrypt -> POST to Python AI -> parse."
	@echo "  Env vars:"
	@echo "    CREDENTIAL_EXTRACTION_BENCH_COUNT    (default: 3)"
	@echo "    CREDENTIAL_EXTRACTION_BENCH_COUNT    (default: 3)"
	@echo "    CREDENTIAL_EXTRACTION_BENCH_CSV   (set to 1 to enable CSV output)"
	@echo "    CREDENTIAL_EXTRACTION_BENCH_PROFILE  (set to 1 to enable CPU+mem+trace profiling)"
	@echo "  Example: make credential-extraction-benchmark CREDENTIAL_EXTRACTION_BENCH_CSV=1"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up         - Start all services with Docker"
	@echo "  make docker-up-build   - Rebuild and start all services"
	@echo "  make docker-down       - Stop and remove all containers"
	@echo "  make docker-restart    - Restart all services with rebuild"
	@echo "  make docker-migrate-up    - Run database migrations (Docker)"
	@echo "  make docker-migrate-down  - Rollback database migrations (Docker)"
	@echo "  make docker-logs       - View container logs"
	@echo "  make docker-ps         - List running containers"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean            - Clean build artifacts and Docker resources"

check-env:
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "error: $(ENV_FILE) file not found."; \
		exit 1; \
	fi

check-env-docker:
	@if [ ! -f .env.docker ]; then \
		echo "error: .env.docker file not found."; \
		exit 1; \
	fi

clean:
	@if [ -f bin/credchain ]; then rm bin/credchain; fi
	docker compose down -v 2>/dev/null || true

build:
	mkdir -p bin
	go build -o bin/credchain main.go

serve:
	go run main.go serve --env $(ENV_FILE)

dev: check-env
	@if ! command -v air &> /dev/null; then \
		echo "error: air is not installed. Run: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi
	air -c .air.toml

migrate-up:
	go run main.go migrate up --env $(ENV_FILE)

migrate-down:
	go run main.go migrate down --env $(ENV_FILE)

migrate-up-mongo:
	go run main.go migrate-mongo up --env $(ENV_FILE)

migrate-down-mongo:
	go run main.go migrate-mongo down --env $(ENV_FILE)

init-super-admin:
	go run main.go init-super-admin --env $(ENV_FILE)

get-google-id-token:
	go run main.go get-google-id-token --env $(ENV_FILE)

seed:
	go run main.go seed --env $(ENV_FILE)

seed-chain:
	go run main.go seed-chain --env $(ENV_FILE)

CREDENTIAL_EXTRACTION_BENCH_COUNT ?= 3
CREDENTIAL_EXTRACTION_BENCH_DIRECTORY ?= benchmarks/credential-extraction
CREDENTIAL_EXTRACTION_BENCH_CSV ?=
CREDENTIAL_EXTRACTION_BENCH_PROFILE ?=

credential-extraction-benchmark: check-env
	mkdir -p $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)
	@FLAGS="" && \
	if [ -n "$(CREDENTIAL_EXTRACTION_BENCH_CSV)" ]; then FLAGS="$$FLAGS --output $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/results.csv"; fi && \
	if [ -n "$(CREDENTIAL_EXTRACTION_BENCH_PROFILE)" ]; then FLAGS="$$FLAGS --cpuprofile $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/cpu.prof --memprofile $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/mem.prof --trace $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/trace.out"; fi && \
	go run main.go credential-extraction-benchmark \
		--count $(CREDENTIAL_EXTRACTION_BENCH_COUNT) \
		$$FLAGS \
		--env $(ENV_FILE)

docker-up-build: check-env-docker
	docker compose up -d --build

docker-up: check-env-docker
	docker compose up -d

docker-down:
	docker compose down

docker-restart: docker-down docker-up-build

docker-migrate-up: docker-check-backend-healthy
	docker compose exec backend ./server migrate up

docker-migrate-down: docker-check-backend-healthy
	docker compose exec backend ./server migrate down

docker-logs:
	docker compose logs -f

docker-ps:
	docker compose ps

docker-clean-data:
	rm -rf docker/postgres/data/* docker/mongo/data/*

docker-check-backend-healthy:
	@echo "waiting for backend to be healthy..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker compose ps backend | grep -q "(healthy)"; then \
			echo "backend is healthy"; \
			exit 0; \
		fi; \
		echo "waiting... ($$i/10)"; \
		sleep 3; \
	done; \
	echo "error: backend did not become healthy in time"; \
	exit 1

docker-fresh:
	@make docker-down
	@make clean
	@make docker-up-build
	@make docker-ps
	@make docker-migrate-up

BACKUP_TIMESTAMP ?= $(shell date +%Y%m%d_%H%M%S)

docker-backup:
	@echo "Backing up Postgres..."
	docker compose exec backend pg_dump -Fc -U root -h postgres credchain > docker/backups/postgres_$(BACKUP_TIMESTAMP).dump
	@echo "Backing up MongoDB..."
	docker compose exec backend mongodump --uri="mongodb://$(grep MONGO_INITDB_ROOT_USERNAME .env.docker | cut -d= -f2):$(grep MONGO_INITDB_ROOT_PASSWORD .env.docker | cut -d= -f2)@mongo:27017" --archive > docker/backups/mongo_$(BACKUP_TIMESTAMP).archive
	@echo "Backing up credential files..."
	docker compose exec backend tar czf /backups/credentials_$(BACKUP_TIMESTAMP).tar.gz -C $$(grep CREDENTIAL_FILE_STORAGE_PATH .env.docker | cut -d= -f2 || echo "credentials") .
	@echo "manifest" > docker/backups/manifest_$(BACKUP_TIMESTAMP).txt
	@echo "---------" >> docker/backups/manifest_$(BACKUP_TIMESTAMP).txt
	@echo "postgres: postgres_$(BACKUP_TIMESTAMP).dump" >> docker/backups/manifest_$(BACKUP_TIMESTAMP).txt
	@echo "mongo: mongo_$(BACKUP_TIMESTAMP).archive" >> docker/backups/manifest_$(BACKUP_TIMESTAMP).txt
	@echo "credentials: credentials_$(BACKUP_TIMESTAMP).tar.gz" >> docker/backups/manifest_$(BACKUP_TIMESTAMP).txt
	@echo "Backup complete: docker/backups/manifest_$(BACKUP_TIMESTAMP).txt"

docker-restore:
	@echo "Restoring from backup: $(BACKUP_TIMESTAMP)..."
	@test -n "$(BACKUP_TIMESTAMP)" || (echo "error: set BACKUP_TIMESTAMP, e.g. BACKUP_TIMESTAMP=20260709_120000 make docker-restore" && exit 1)
	@echo "Restoring Postgres..."
	docker compose exec -T backend pg_restore -Fc -U root -h postgres -d credchain --clean < docker/backups/postgres_$(BACKUP_TIMESTAMP).dump
	@echo "Restoring MongoDB..."
	docker compose exec -T backend mongorestore --uri="mongodb://$(grep MONGO_INITDB_ROOT_USERNAME .env.docker | cut -d= -f2):$(grep MONGO_INITDB_ROOT_PASSWORD .env.docker | cut -d= -f2)@mongo:27017" --archive --drop < docker/backups/mongo_$(BACKUP_TIMESTAMP).archive
	@echo "Restoring credential files..."
	docker compose exec backend tar xzf /backups/credentials_$(BACKUP_TIMESTAMP).tar.gz -C $$(grep CREDENTIAL_FILE_STORAGE_PATH .env.docker | cut -d= -f2 || echo "credentials")
	@echo "Restore complete."