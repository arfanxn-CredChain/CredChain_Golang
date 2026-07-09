# CredChain — Go Backend API

Production backend for the CredChain decentralized credential platform.

## Stack

Go 1.25 · Gin v1.12 · GORM v1.31 (PostgreSQL 16 + SQLite for tests) · MongoDB 8.0 · go-ethereum v1.17 · Uber FX · Cobra CLI · River job workers

## Quick Start

```bash
cp .env.example .env
# Fill in required vars (see .env.example for full list)
make serve     # or make dev for hot reload
```

The API starts on port 8080. Requires PostgreSQL, MongoDB, and a local Ethereum node (Hardhat/Anvil).

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

| Command | Purpose |
|---|---|
| `make serve` | Start API server |
| `make dev` | Start with hot reload (requires air) |
| `make test` | Run all tests |
| `make migrate-up` | Run PostgreSQL migrations |
| `make migrate-up-mongo` | Create MongoDB indexes |
| `make init-super-admin` | Create super admin (local CLI) |
| `make docker-init-super-admin` | Create super admin (Docker) |
| `make seed` | Seed development data (local CLI) |
| `make docker-seed` | Seed development data (Docker) |
| `make seed-chain` | Register roles on-chain (local CLI) |
| `make docker-seed-chain` | Register roles on-chain (Docker) |
| `docker compose up -d` | Start infrastructure containers |

## Related Docs

- [AGENTS.md](AGENTS.md) — Full architecture, patterns, conventions, deployment
- [ROLE.md](ROLE.md) — Role hierarchy, authorization matrix, user policy rules
- [CREDENTIAL.md](CREDENTIAL.md) — Credential lifecycle, verification pipeline, storage
