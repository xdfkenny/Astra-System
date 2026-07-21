# Kiosk Packages — Voicebank Architecture

Astra-System follows a **Studio + Voicebank** architecture inspired by
Vocaloid Bench: the **Astra-System platform** (the studio) provides the
infrastructure engine, and **individual kiosk applications** (voicebanks)
connect to it via APIs.

## The Model

```
┌─────────────────────────────────────────────────────┐
│                  ASTRA-SYSTEM (Studio)               │
│                                                     │
│  PostgreSQL · Redis · NATS JetStream                │
│  Gateway API (:8080) · Cart · Order · Payment       │
│  Inventory · Sync · WebAuthn · Admin GraphQL        │
│                                                     │
│  One install. Deploys everything via Docker.        │
└──────────────────┬──────────────────────────────────┘
                   │
                   │  HTTP API (JWT auth)
                   │
     ┌─────────────┼─────────────┬──────────────────┐
     ▼             ▼             ▼                  ▼
┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐
│ Andino  │  │  Future  │  │  Future  │  │   Your Kiosk    │
│ Kiosk   │  │  Client  │  │  Client  │  │   (you build)   │
│ :3000   │  │  :4000   │  │  :5000   │  │   :6000         │
├─────────┤  ├─────────┤  ├─────────┤  ├─────────────────┤
│ Nuxt 3  │  │  React   │  │  Flutter │  │  Any framework  │
│ Nitro   │  │  SPA     │  │  Mobile  │  │                 │
│ Andino  │  │  Custom  │  │  POS     │  │  Docker image    │
│ API     │  │  API     │  │  API     │  │  + compose file  │
└─────────┘  └─────────┘  └─────────┘  └─────────────────┘
```

## Why This Architecture

| Benefit | Explanation |
|---|---|
| **One backend, many frontends** | The studio handles all heavy lifting — DB, auth, messaging, payment orchestration. Each kiosk only needs to render UI and call APIs. |
| **Independent deployment** | Kiosks can be updated, restarted, or replaced without touching the studio. |
| **Technology freedom** | Each kiosk can use whatever framework fits the client: Nuxt for web-first, React for SPA, Flutter for mobile, etc. |
| **No vendor lock-in** | Clients aren't tied to the Astra-System frontend — they can build their own. |
| **Simple packaging** | Each kiosk is just a `Dockerfile` + `docker-compose.<name>.yml` + an install script. |

## Existing Voicebanks

| Package | Location | Description |
|---|---|---|
| **Andino Kiosk** | `packages/andino-kiosk/` | Self-service cafeteria kiosk for Meriandes schools (Nuxt 3 + Andino API) |

## Creating a New Voicebank

### Step 1: Create the package directory

```
packages/<your-kiosk>/
├── Dockerfile
├── docker-compose.<name>.yml
├── .env.<name>.example
├── install.ps1          # Windows installer
├── install.sh           # macOS/Linux installer (optional)
└── README.md
```

### Step 2: Write the Dockerfile

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY .output /app/.output
COPY scripts/start-production.mjs /app/scripts/
EXPOSE 3000
CMD ["node", "scripts/start-production.mjs"]
```

### Step 3: Write the compose override

```yaml
services:
  my-kiosk:
    image: ghcr.io/your-org/my-kiosk:latest
    container_name: astra-my-kiosk
    restart: unless-stopped
    ports:
      - "4000:3000"
    environment:
      API_GATEWAY_URL: http://gateway:8080
    networks:
      - astra-net

networks:
  astra-net:
    external: true
```

### Step 4: Write the install script

```powershell
param([switch]$BuildLocal, [string]$SourcePath)

if ($BuildLocal) {
    docker build -t ghcr.io/your-org/my-kiosk:latest $SourcePath
}

docker compose -f "packages/my-kiosk/docker-compose.my-kiosk.yml" up -d
Write-Host "My Kiosk running at http://localhost:4000"
```

### Step 5: Document the env vars

Create `.env.<name>.example` with all configurable variables.

### Step 6: Document the package

Create `README.md` explaining:
- What the kiosk does
- Required credentials / API keys
- How to install
- How to configure

## Connecting to the Studio

All voicebanks connect to the Astra-System Gateway at:

```
http://gateway:8080   (from within the Docker network)
http://localhost:8080 (from the host machine)
```

The gateway provides:

| Endpoint | Purpose |
|---|---|
| `GET /health` | Health check |
| `GET /v1/menu` | Product catalog |
| `GET /v1/carts/:id` | Cart operations |
| `POST /v1/orders` | Order placement |
| Plus payment, auth, inventory, etc. | |

All endpoints require JWT authentication (except `/health`).

## Installing a Voicebank

Each voicebank has its own install script:

```powershell
# Windows
.\packages\<kiosk>\install.ps1

# macOS/Linux
./packages/<kiosk>/install.sh
```

The script:
1. Creates `.env` from template if missing
2. Builds or pulls the Docker image
3. Starts the container connected to the Astra network
