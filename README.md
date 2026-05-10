# mini-brimble

A small, self-hosted deployment platform inspired by Railway/Render. Point it at a public GitHub repo and it will clone, build a container image via [Railpack](https://railpack.com), run it, and reverse-proxy a live URL to it — all on your laptop.

## Stack

- **Server**: Go 1.26 · Gin · GORM
- **Client**: React 19 · Vite · TanStack Router & Query · Tailwind 4
- **Database**: PostgreSQL 17
- **Builder**: Railpack (CLI) + BuildKit (container)
- **Reverse proxy**: Caddy v2 (configured live via its admin API)
- **Orchestrator**: Docker, mounted via socket from the host

## Architecture

```
                                              ┌─────────────────────────┐
                                              │  user's browser         │
                                              │  http://<id>.localhost  │
                                              └────────────┬────────────┘
                                                           │
                                                  ┌────────▼────────┐
                                                  │  caddy :80      │
                                                  │  (admin :2019)  │
                                                  └────────┬────────┘
                                                           │ reverse_proxy
                                                           │ host.docker.internal:N
                                                           ▼
                              ┌─────────────────┐    ┌─────────────────────┐
                              │  client :3000   │    │ deployment-<uuid>   │
                              │  (vite + nginx) │    │ container (port N)  │
                              └────────┬────────┘    └─────────────────────┘
                                       │                       ▲
                                  REST / SSE                   │  docker run
                                       │                       │
                                  ┌────▼─────────────────┐     │
                                  │  server :8080        │─────┘
                                  │  (Go / Gin)          │
                                  └────┬───┬─────────┬───┘
                                       │   │         │
                       gorm/postgres   │   │         │   admin api (PUT/DELETE)
                                       │   │         └──────► caddy
                                       │   │
                                       │   └─ buildkit  ──► moby/buildkit (unix socket)
                                       │   └─ railpack  ──► /usr/local/bin/railpack
                                       │   └─ docker    ──► /var/run/docker.sock (host)
                                       ▼
                                  postgres :5432
```

### How a deployment flows

1. **Create** — `POST /api/v1/deployments` with a `github_url`. The server inserts a row (`status=pending`) and returns immediately with a deployment ID. A goroutine picks up the work.
2. **Clone** — `git clone --depth 1` into a temp workspace inside the server container.
3. **Build** — `railpack build <projectDir> --name <imageName>`. Railpack detects the language, spawns a BuildKit job (over the shared `/run/buildkit/buildkitd.sock` volume), and on success calls `docker load` to register the image with the host daemon.
4. **Inspect** — the server inspects the resulting image's `EXPOSE` directive. If present, that port is the container port; otherwise it defaults to 8080.
5. **Run** — the server allocates a free host port, then `docker run`s the image with:
   - `PORT=<containerPort>` env (so frameworks like Next.js / Express / Flask listen on the right port)
   - port binding `0.0.0.0:<hostPort> → <containerPort>/tcp`
6. **Route** — the server `PUT`s a `reverse_proxy` route at index `0` of Caddy's `srv0/routes` array, with a host matcher of `<id>.localhost` and an upstream of `host.docker.internal:<hostPort>`. Inserting at index 0 places the deployment route before the static `caddy ready` catch-all, so Caddy matches it first.
7. **Live** — `http://<id>.localhost` now reaches the container.

### How stop works

`DELETE /api/v1/deployments/:id` does the following, in order:

1. If a build goroutine is still in flight, its context is cancelled — `exec.CommandContext` kills the railpack/git subprocess. The handler waits up to 5s for the goroutine to unwind.
2. Caddy route deleted via `DELETE /id/<routeID>` (404 is fine — idempotent).
3. Container stopped + force-removed (`containerd/errdefs.IsNotFound` is treated as success — idempotent).
4. Deployment row deleted from Postgres (`logs` cascade-delete).

Each step is best-effort: a failure in one step is logged to the deployment's log stream but doesn't abort the others. The DB row always gets removed.

### How log streaming works

- The server writes every log line to Postgres (`logs` table) **and** broadcasts an event `{id, message, created_at}` over an in-memory `logstream.Hub`.
- `GET /api/v1/deployments/:id/logs` returns the historical snapshot.
- `GET /api/v1/deployments/:id/logs/stream` is a Server-Sent Events endpoint that streams new events as they fire.
- The client merges both sources through a `Set<id>` so events appearing in both (race between snapshot and subscribe) are shown once.

## Setup

### Prerequisites

- Docker Engine + Docker Compose v2
- A Linux host with access to `/var/run/docker.sock` (the server container needs to talk to your Docker daemon to start deployment containers)
- A modern browser. `*.localhost` resolves to `127.0.0.1` on Firefox, Chrome, and most modern OSes — no `/etc/hosts` edits required.

### First run

```bash
git clone <this repo>
cd mini-brimble
docker compose up -d --build
```

That brings up five services:

| service   | port  | purpose                                              |
| --------- | ----- | ---------------------------------------------------- |
| `client`  | 3000  | React UI                                             |
| `server`  | 8080  | Go API + deployment orchestrator                     |
| `db`      | 5432  | Postgres (data persisted in the `postgres_data` volume) |
| `caddy`   | 80, 2019 | Reverse proxy for deployment routes (admin on 2019) |
| `buildkit` | —    | BuildKit daemon, shared with the server over a volume |

Then open <http://localhost:3000>, paste a public GitHub URL, and click **Deploy**.

### Environment variables

All settable via env on the `server` service in `docker-compose.yml`:

| key                         | default                  | meaning                                                    |
| --------------------------- | ------------------------ | ---------------------------------------------------------- |
| `PORT`                      | `8080`                   | server listen port                                         |
| `APP_BASE_URL`              | `localhost`              | base domain for deployment routes (`<id>.<APP_BASE_URL>`)  |
| `DEPLOYMENT_UPSTREAM_HOST`  | `host.docker.internal`   | what Caddy dials to reach the deployment container         |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | — | Postgres connection                  |
| `DOCKER_SOCKET_PATH`        | `/var/run/docker.sock`   | Docker daemon socket                                       |
| `CADDY_HOST` / `CADDY_PORT` | `caddy` / `2019`         | Caddy admin API                                            |
| `BUILDKIT_HOST`             | `unix:///run/buildkit/buildkitd.sock` | BuildKit daemon endpoint                  |

The client reads `VITE_API_BASE_URL` (default `http://localhost:8080/api/v1`) at build time.

## Repo layout

```
mini-brimble/
├── client/                 # React + Vite UI
│   └── src/
│       ├── api/            # Fetch wrappers
│       ├── components/     # DeployModal, DeploymentCard, LogViewer, etc.
│       └── routes/         # TanStack Router pages
├── server/
│   ├── cmd/server/         # main.go
│   └── internal/
│       ├── api/            # Gin handlers
│       ├── caddy/          # admin API client
│       ├── config/         # env loader
│       ├── database/       # GORM setup + migrations
│       ├── deployment/     # orchestration service (clone → build → run → route)
│       ├── deploymentstore # GORM repository for deployments
│       ├── docker/         # moby/moby client wrapper
│       ├── logstore/       # GORM repository for logs
│       ├── logstream/      # in-memory pub/sub for live log SSE
│       ├── models/         # GORM models (Deployment, LogEntry)
│       ├── railpack/       # exec wrapper around the railpack CLI
│       └── router/         # gin route registration + CORS
├── infra/caddy/Caddyfile   # bootstraps caddy with empty srv0 + catch-all
├── docker-compose.yml      # full local stack
└── app/                    # tiny sample Go app for end-to-end testing
```

## API reference

| method | path                                      | description                              |
| ------ | ----------------------------------------- | ---------------------------------------- |
| GET    | `/health`                                 | liveness probe                           |
| GET    | `/ready`                                  | readiness probe (pings the DB)           |
| POST   | `/api/v1/deployments`                     | create a deployment (`{"github_url":""}`) |
| GET    | `/api/v1/deployments`                     | list all deployments                     |
| GET    | `/api/v1/deployments/:id`                 | fetch one deployment                     |
| DELETE | `/api/v1/deployments/:id`                 | stop + remove a deployment               |
| GET    | `/api/v1/deployments/:id/logs`            | historical logs (JSON)                   |
| GET    | `/api/v1/deployments/:id/logs/stream`     | live log stream (SSE)                    |

All JSON responses follow `{ success, message, data? }`.

## Known limitations

- **Single-host** — everything runs on one Docker daemon. There is no scheduler, no replicas, no zero-downtime.
- **No auth** — the API and the Caddy admin endpoint are wide open. Don't expose this beyond `localhost`.
- **Public repos only** — no SSH keys, no PAT support.
- **Hardcoded port fallback** — if an app neither reads `PORT` nor declares `EXPOSE`, the system maps `8080` and silently fails when the app isn't listening there.
- **No image/build cache GC** — successful builds leave images and cloned workspaces; failed builds clean up the workspace but may leave layer cache in BuildKit.
- **CORS allowlist is hardcoded** to `http://localhost:3000`.
- **First build is slow** — BuildKit pulls the railpack base images (`ghcr.io/railwayapp/railpack-builder` and `railpack-runtime`) the first time. Subsequent builds reuse them.

## Walkthrough: deploying a Next.js app

1. Open <http://localhost:3000>, click **New Deployment**, paste a public Next.js repo URL, submit.
2. Watch the **Build** tab — you'll see the railpack output: detected Node, `npm ci`, `npm run build`, image export.
3. Once the status flips to **Running**, the **Runtime** tab takes over and the live URL appears (e.g. `http://b18ab64b-….localhost`).
4. Click it — Caddy proxies to your container; Next.js, having been told `PORT=8080`, is listening there.
5. Click **Stop** when done — the container is removed, the Caddy route is removed, and the row is deleted.

Behind the scenes, the same flow works for Express, Fastify, Flask (via gunicorn), or any image that respects `PORT` or declares `EXPOSE`.
