# Run CredChain Locally — Implementation Plan

**Goal:** Bring the full CredChain stack up on the host (Hardhat + deployed contracts, Python AI in Docker, Go backend on the host, React frontend) with PostgreSQL + MongoDB already running in Docker, and verify no PID leaks memory across a working session.

**Architecture:** Chain, AI, and databases start first; the Go backend runs on the host via `make serve`/`make dev` and consumes them; the React dev server proxies `/api` to the backend. A monitoring loop tracks RSS for every process.

**Tech Stack:** Hardhat 2.28 (Solidity 0.8.24), FastAPI + uvicorn (single worker), Go 1.25.1 + Gin + Uber FX, React 19 + Vite 7, PostgreSQL 15, MongoDB 8.0, Docker Compose.

**Status:** Executed 2026-06-08. All services verified healthy.

---

## Pre-flight checks

1. Databases reachable:
   - `docker exec postgres pg_isready -U root -d credchain`
   - `docker exec mongo mongosh "mongodb://root:root@localhost:27017/credchain?authSource=admin" --quiet --eval 'db.runCommand({ ping: 1 })'`
2. `backend` Docker network exists: `docker network inspect backend` (else `docker network create backend`)
3. Tooling: `npm ls hardhat` in `CredChain_Solidity/`; `go version`; optional `air`
4. React deps: `node_modules/` present in `CredChain_React/`

> Note: local `psql`/`mongosh` CLIs were absent in this run — verified DB connectivity through the containers instead.

---

## Step 1 — Local chain (CredChain_Solidity)

1. Terminal: `npx hardhat node` from `CredChain_Solidity/` (binds `http://127.0.0.1:8545`; account #0 deploys, #1 is SuperAdmin)
2. Confirm `CredChain_Solidity/.env`:
   - `PRIVATE_KEY=0xac09...ff80` (account #0 — deployer/relayer)
   - `POLYGON_RPC_URL=http://127.0.0.1:8545`
   - `INITIAL_SUPER_ADMIN_WALLET_ADDRESS=0x7099...79C8` (account #1)
3. Second terminal: `npx hardhat run scripts/deploy.ts --network polygon`; capture Authority + Registry addresses

**Gotcha:** Go `.env` pins `AUTHORITY_CONTRACT=0xe7f1...0512` and `REGISTRY_CONTRACT=0x9fE4...a6e0`. These are the deterministic addresses a fresh node produces (Config→Authority→Registry from nonce 0–2). In this run they matched the deploy output — no Go `.env` change needed.

---

## Step 2 — Python AI service (Docker)

1. Ensure `CredChain_Python/.env.docker` has `GEMINI_API_KEY`, `HF_TOKEN`, `API_KEY` (64-char hex). If `API_KEY` empty: `make docker-generate-api-key`.
2. `make docker-up-build` from `CredChain_Python/` (attaches to `backend` network, exposes `127.0.0.1:8081`). If the container already exists/stopped: `docker start credchain-python`.
3. Sanity: `curl http://127.0.0.1:8081/health` → expect `code: 500900`. First boot downloads EmbeddingGemma from HuggingFace (slow).

---

## Step 3 — Sync Go backend env

`CredChain_Golang/.env`:

| Var | Value |
|---|---|
| `POSTGRES_DSN` | `postgres://root:root@localhost:5432/credchain?sslmode=disable` (set) |
| `MONGO_URI` | `mongodb://root:root@localhost:27017` (set) |
| `RPC_URL` | `http://127.0.0.1:8545` (set) |
| `AUTHORITY_CONTRACT` / `REGISTRY_CONTRACT` | from Step 1 deploy output |
| `PYTHON_AI_API_KEY` | copy from Python `.env.docker` `API_KEY` |
| `PYTHON_AI_BASE_URL` | defaults to `http://localhost:8081` (no change needed) |
| `WALLET_ENCRYPTION_KEY` | already 32 bytes |
| `JWT_SECRET`, `GOOGLE_CLIENT_ID/_SECRET/_REDIRECT_URI` | already set |

---

## Step 4 — Initialize database

From `CredChain_Golang/`:

1. `make migrate-up` — Postgres schema + River tables (no-op if already migrated)
2. `make migrate-up-mongo` — Mongo indexes for `credential_extractions`, `credential_verifications`
3. `make init-super-admin` — verifies on-chain SuperAdmin role, creates DB row (idempotent; aborts if live SuperAdmin exists)

---

## Step 5 — Run Go server

From `CredChain_Golang/`: `make serve` (or `make dev` for hot reload).
Verify: `curl http://localhost:8080/api/health` → `200`, code `100000`.

---

## Step 6 — Run React frontend

From `CredChain_React/`:

1. `npm run dev` — Vite on `:5173`, proxies `/api` → `:8080`
2. Verify:
   - `curl -I http://localhost:5173/` → `200`
   - `curl http://localhost:5173/api/health` → code `100000` (confirms proxy reaches backend)

**Env (`.env.local`, already set):** `VITE_GOOGLE_CLIENT_ID`, `VITE_API_BASE_URL=/api`, `VITE_API_PROXY=http://localhost:8080`.

**CORS:** backend `GIN_CORS_ALLOW_ORIGINS=http://localhost:5173` (correct; not `*` while credentials enabled).

---

## Step 7 — Memory-leak monitoring

Risk surface: Go FX singletons (GORM pool, ethclient, rate-limiter maps, River workers), Hardhat in-memory block/state accumulation, Python EmbeddingGemma RAM, Vite watcher file handles.

### 7.1 Baselines & thresholds

| Process | Idle RSS | Warn | Reset |
|---|---|---|---|
| Go server | ~80–150 MB | > 400 MB sustained 10 min | > 800 MB or >5%/min for 5 min |
| Python container | ~1.5–2.5 GB after load | > 3.5 GB | > 5 GB |
| Hardhat node | ~150–300 MB | > 1 GB | > 2 GB or after 50+ tx batches |
| Vite dev server | ~200–400 MB | > 800 MB sustained | > 1.5 GB or watcher EMFILE |

Captured baselines this run: Go ~52 MB, Hardhat ~40 MB, Python ~792 MB, Postgres ~29 MB, Mongo ~189 MB.

### 7.2 Sampling loop

```bash
while true; do
  echo "=== $(date +%H:%M:%S) ==="
  ps -o pid,rss,command -p $(pgrep -f 'main.go serve' | tail -1) 2>/dev/null
  ps -o pid,rss,command -p $(pgrep -f 'hardhat node' | tail -1) 2>/dev/null
  ps -o pid,rss,command -p $(pgrep -f 'vite' | tail -1) 2>/dev/null
  docker stats --no-stream --format 'python mem={{.MemUsage}}' credchain-python
  sleep 30
done | tee /tmp/credchain-logs/mem-watch.log
```

Sustained upward slope with no plateau = leak signal.

### 7.3 Go deep checks (on growth)

1. Mount pprof at `/debug/pprof` (dev only; gate on `GIN_MODE != "release"`).
2. Diff two heap profiles 5 min apart:
   ```bash
   curl -s http://localhost:8080/debug/pprof/heap > logs/heap-1.pb.gz
   sleep 300
   curl -s http://localhost:8080/debug/pprof/heap > logs/heap-2.pb.gz
   go tool pprof -base logs/heap-1.pb.gz logs/heap-2.pb.gz
   ```
   Suspects: GORM pool, ethclient subscriptions, rate-limiter maps, in-flight HTTP buffers.
3. Goroutines: `curl -s http://localhost:8080/debug/pprof/goroutine?debug=2 | head -200`. Linear growth parked on `chan receive`/`select`/`WaitGroup.Wait` = leak.
4. One race-mode run: `go run -race main.go serve --env .env` (don't keep long-term; 5–10× RSS overhead).

If RSS climbs but heap stays flat → goroutine leak, not allocation.

### 7.4 Python container

Optional hard cap in `CredChain_Python/docker-compose.yml`:
```yaml
services:
  credchain-python:
    mem_limit: 6g
    memswap_limit: 6g
```
Watch: `docker stats credchain-python`. Plateau expected after a few `/extract` calls; monotonic climb past 3.5 GB → restart container, report upstream (don't just raise the cap).

### 7.5 Hardhat node

Recycle on schedule (retains every block + state diff). On restart: redeploy → `init-super-admin` → reset DBs (`make migrate-down && make migrate-up`, same for `-mongo`). Early restart if RSS > 2 GB.

### 7.6 Vite watcher

Restart if RSS > 1.5 GB or `EMFILE` appears in logs (watcher handle accumulation).

### 7.7 Reset when leak confirmed

1. Capture artifacts:
   ```bash
   mkdir -p logs/leak-$(date +%s)
   cp logs/heap-*.pb.gz logs/mem-watch.log logs/leak-$(date +%s)/ 2>/dev/null
   docker logs credchain-python > logs/leak-$(date +%s)/python.log 2>&1
   ```
2. Down in reverse start order: Go (Ctrl-C) → Vite (Ctrl-C) → Python (`make docker-down`) → Hardhat (Ctrl-C).
3. Restart from Step 1.

---

## Run order summary

```
Terminal 1 (Solidity):   npx hardhat node
Terminal 2 (Solidity):   npx hardhat run scripts/deploy.ts --network polygon
                         → reconcile Go .env addresses
Terminal 3 (Python):     make docker-up-build
Terminal 4 (Golang):     make migrate-up && make migrate-up-mongo
                         make init-super-admin
                         make serve
Terminal 5 (React):      npm run dev
Terminal 6 (Watch):      sampler loop (7.2)
```

## Risks

1. Hardhat restart resets state → always redeploy + `init-super-admin` + reset DBs.
2. Stale Postgres/Mongo after chain restart → on-chain reads empty for DB-referenced users.
3. Empty `PYTHON_AI_API_KEY` in Go while Python enforces it → 401 on AI endpoints.
4. Missing `backend` Docker network → Python compose fails.
5. `GIN_CORS_ALLOW_CREDENTIALS=true` requires non-`*` origins.
6. `WALLET_ENCRYPTION_KEY` must stay 32 bytes.
7. Keep `/debug/pprof` host-only and behind `GIN_MODE != "release"`.
8. Vite EMFILE → restart vite.

## Verification checklist

- [x] Hardhat node on `:8545`, contracts deployed (addresses match Go `.env`)
- [x] Python healthy (`/health` → 500900)
- [x] Go backend healthy (`/api/health` → 100000)
- [x] React healthy (`/` → 200; `/api/health` via proxy → 100000)
- [x] Baselines captured; sampler documented

## Next steps

1. Open http://localhost:5173
2. Log in via Google (frontend) or `make get-google-id-token` (Postman)

---

*Logs: /tmp/credchain-logs/{hardhat,golang,react}.log*
