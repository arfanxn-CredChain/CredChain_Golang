# CredChain — Go Backend API

Production backend for the CredChain decentralized credential platform.

## Stack

Go 1.25 · Gin v1.12 · GORM v1.31 (PostgreSQL 16 + SQLite for tests) · MongoDB 8.0 · go-ethereum v1.17 · Uber FX · Cobra CLI · River job workers

## Quick Start

```bash
cp .env.example .env
# Fill in required vars (see .env.example for full list)
make dev-up    # infra + Anvil + Python in Docker, deploy contracts, migrate, seed
make dev       # run the API on host with hot reload
```

The API starts on port 8080. `make dev-up` brings up the PostgreSQL, MongoDB, and Anvil containers it depends on. For the full-Docker stack instead, use `make up`.

## Project Structure

```
CredChain_Golang/
├── cmd/              # Cobra CLI commands
├── config/           # Env config via Pydantic-style loader
├── domain/           # Domain entities, error codes, query types
├── feature/          # Business logic per domain (auth, user, credential, overview, meta)
├── infrastructure/   # Database, HTTP router, chain bindings, AI client, jobs, i18n, logging
├── locales/          # en.json + id.json
├── tests/            # Test helpers, fixtures, gintest, mocks
└── docker/           # Docker Compose: postgres, mongo, hardhat, nginx
```

## Key Commands

This repo is the **orchestrator** for the whole monorepo. Run `make help` for the full list.

| Command | Purpose |
|---|---|
| `make up` | Full stack in Docker (contracts, migrate, super-admin, all services) |
| `make down` | Stop all containers |
| `make fresh` | Wipe volumes + uploads, then `up` |
| `make logs` | Follow backend/infra logs |
| `make dev-up` | Local hybrid: infra + Anvil + Python in Docker, deploy contracts, migrate, seed |
| `make dev-fresh` | `dev-up` after wiping local infra data + chain (clean reset + reseed) |
| `make dev` | Run API on host with hot reload (requires air) |
| `make test` | Run all tests |
| `make lint` | `go vet` + `gofmt` |
| `make backup` / `make restore` | Dump / restore Postgres + Mongo + uploads (`BACKUP=<ts>`) |

Migrate / seed / seed-chain / init-super-admin run automatically inside `up`/`dev-up`. For a standalone run, call the CLI: `go run main.go <cmd> --env .env`.

## Related Docs

- [AGENTS.md](AGENTS.md) — Full architecture, patterns, conventions, deployment
- [ROLE.md](ROLE.md) — Role hierarchy, authorization matrix, user policy rules
- [CREDENTIAL.md](CREDENTIAL.md) — Credential lifecycle, verification pipeline, storage
