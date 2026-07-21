# Andino Kiosk — Voicebank Package

The Andino Kiosk is a **self-service cafeteria kiosk** built for Meriandes
schools. It is a **voicebank** that runs on top of the **Astra-System studio**
— the platform provides the infrastructure, and individual kiosk apps connect
via APIs.

## Architecture

```
┌──────────────────────────────────────────────────┐
│                  Astra-System                     │
│                 (The Studio)                      │
│                                                   │
│  ┌─────────┐ ┌──────────┐ ┌────────┐             │
│  │PostgreSQL│ │  Redis   │ │  NATS  │             │
│  │   :5432  │ │  :6379   │ │ :4222  │             │
│  └─────────┘ └──────────┘ └────────┘             │
│  ┌─────────┐ ┌──────────┐ ┌────────────────────┐ │
│  │ Gateway │ │  Cart    │ │ Payment-Orch.  ... │ │
│  │  :8080  │ │  :8081   │ │                    │ │
│  └─────────┘ └──────────┘ └────────────────────┘ │
└──────────────────────────────────────────────────┘
           ▲                ▲
           │                │
    ┌──────┴──────┐  ┌──────┴──────┐
    │   Andino    │  │  Future     │
    │   Kiosk     │  │  Kiosks...  │
    │   :3000     │  │             │
    │             │  │             │
    │ Nuxt 3      │  │             │
    │ Nitro API   │  │             │
    │ File-based  │  │             │
    │ storage     │  │             │
    └─────────────┘  └─────────────┘
```

**Like Vocaloid Bench**, the **studio** (Astra-System) provides the engine
(database, message broker, API gateway, payment orchestration), and each
**voicebank** (Andino Kiosk, FutureClient Kiosk, etc.) is a standalone
application that connects to the studio via its API.

## How It Works

### Studio (Astra-System)
- Docker microservices (Go, TypeScript, Python)
- PostgreSQL for persistent data
- NATS JetStream for event-driven messaging
- Redis for caching and session state
- JWT-authenticated API gateway on `:8080`
- Deployed via the main Astra-System installer

### Voicebank (Andino Kiosk)
- Nuxt 3 + Nitro production build
- File-based JSON storage (no database needed)
- Connects to **Andino API** (`https://andinoapp.com`) for:
  - Product catalog (`GET /api/pos/products`)
  - User authentication (`GET /api/auth/user`)
- Self-contained HTTP server on `:3000`
- CLI installable via `packages/andino-kiosk/install.ps1`

## Creating Your Own Voicebank

Any application that connects to the Astra-System Gateway(`:8080`) is a
voicebank. The minimum requirements:

1. **Package it as a Docker image** with a `Dockerfile`
2. **Create a compose override** (`docker-compose.<name>.yml`) that:
   - Adds your service
   - Connects to the `astra-net` network
   - Exposes the ports you need
3. **Provide an install script** (`.ps1` / `.sh`) that users run
4. **Document your env vars** in `.env.<name>.example`
5. **Place everything in** `packages/<your-kiosk>/`

### Minimal Example

```
packages/my-kiosk/
├── Dockerfile
├── docker-compose.my-kiosk.yml
├── .env.my-kiosk.example
├── install.ps1          # Windows
├── install.sh           # macOS/Linux
└── README.md
```

### Docker Compose Override Template

```yaml
services:
  my-kiosk:
    image: ghcr.io/your-org/astra-system/my-kiosk:latest
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

The key is connecting to the `astra-net` external network so your kiosk can
reach the gateway at `http://gateway:8080`.

## Installing the Andino Kiosk

### Prerequisites
- Running Astra-System stack (deployed via `Astra-System-Setup.exe`)
- Andino API credentials (from andinoapp.com)

### Quick Start

```powershell
# 1. Create your .env file
copy packages\andino-kiosk\.env.andino.example packages\andino-kiosk\.env.andino

# 2. Edit .env.andino with your credentials
notepad packages\andino-kiosk\.env.andino

# 3. Run the installer
.\packages\andino-kiosk\install.ps1
```

The kiosk will be available at `http://localhost:3000`.

### Building from Source

```powershell
.\packages\andino-kiosk\install.ps1 -BuildLocal -SourcePath "D:\selfservice-cafeteria"
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ANDINO_PRODUCT_MODE` | `api` | `api` = live Andino, `local` = mock data |
| `ANDINO_BASE_URL` | `https://andinoapp.com` | Andino API base URL |
| `ANDINO_ACCESS_TOKEN` | — | JWT Bearer token for Andino API |
| `ANDINO_SCHOOL_ID` | `9` | School identifier |
| `ANDINO_PROTOTYPE_USER_ID` | — | Demo user ID |
| `ANDINO_PROTOTYPE_PIN` | `123456` | Demo user PIN |
| `ANDINO_ADMIN_CREDENTIALS` | `admin:changeme` | Admin login (user:pin) |
| `ANDINO_ADMIN_SESSION_SECRET` | — | Session signing secret (32+ chars) |
| `ANDINO_POS_ENABLED` | `false` | Enable card payment terminal |
| `ANDINO_PRINTER_ENABLED` | `false` | Enable receipt printer |
| `ANDINO_P2P_ENABLED` | `false` | Enable P2P mesh sync |
| `ANDINO_FACE_ENABLED` | `false` | Enable face recognition |

## Ports

| Port | Service |
|---|---|
| `3000` | Kiosk UI + Nitro API |
| `3001` | P2P Socket.IO (if enabled) |
| `8080` | Astra-System Gateway (from studio) |

## Directory Structure

```
packages/andino-kiosk/
├── Dockerfile                  # Multi-stage Node 22 + Nuxt 3 build
├── docker-compose.andino.yml   # Compose override for Andino
├── .env.andino.example         # Environment template
├── install.ps1                 # Windows install script
└── README.md                   # This file
```
