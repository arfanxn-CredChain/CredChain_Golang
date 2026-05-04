# CredChain - Makefile Commands

ENV_FILE ?= .env
ifneq (,$(wildcard $(ENV_FILE)))
    include $(ENV_FILE)
    export
endif

.PHONY: help check-env check-env-docker clean build serve migrate-up migrate-down init-super-admin \
	docker-migrate-up docker-migrate-down docker-up-build docker-up \
	docker-down docker-restart docker-logs docker-ps docker-fresh \
	docker-clean-data docker-check-backend-healthy

help:
	@echo "CredChain - Available Commands:"
	@echo ""
	@echo "Usage:"
	@echo "  make serve ENV=.env.docker      # Use custom env file"
	@echo ""
	@echo "Local Development:"
	@echo "  make build              - Build the application"
	@echo "  make serve             - Start the application server"
	@echo "  make migrate-up        - Run database migrations up (local)"
	@echo "  make migrate-down      - Rollback database migrations (local)"
	@echo "  make init-super-admin  - Create super admin user (local)"
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

migrate-up:
	go run main.go migrate up --env $(ENV_FILE)

migrate-down:
	go run main.go migrate down --env $(ENV_FILE)

init-super-admin:
	go run main.go init-super-admin --env $(ENV_FILE)

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