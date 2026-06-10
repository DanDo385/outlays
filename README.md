# Outlays

Make any government-spending question answerable in seconds, with a cryptographically
verifiable citation back to the source row. Neutral method is the product: no editorializing,
ever.

- **[CLAUDE.md](./CLAUDE.md)** — mission, hard rules, roadmap, conventions.
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** — normative spec (data model, contract, API,
  decision log).
- **[BUILD_TASKS.md](./BUILD_TASKS.md)** — task sequence and acceptance criteria.

## Layout

```
packages/{contract, adapter-sdk-ts, adapters/*, web}   pnpm workspace (TypeScript)
py/adapter_sdk                                          Python adapter SDK
core/                                                   Go module (orchestrator, api, conformance, anchor)
contracts/                                              Foundry (anchor layer)
deploy/docker-compose.yml                               Postgres 16 + MinIO
data/cofog/  docs/sources/                              research inputs
```

## Quick start (local dev)

```sh
cp .env.example .env
docker compose -f deploy/docker-compose.yml up -d   # Postgres + MinIO
pnpm install && pnpm -r build                        # TypeScript workspace
( cd core && go build ./... )                        # Go core
( cd contracts && forge build )                      # Solidity
```

Apache-2.0 licensed.
