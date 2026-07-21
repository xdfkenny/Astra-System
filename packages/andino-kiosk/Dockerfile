# syntax=docker/dockerfile:1.7
# Andino Kiosk — Self-Service Cafeteria Kiosk for Meriandes
# Nuxt 3 + Nitro node-server production image

ARG NODE_VERSION=22

# ── Build stage ───────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS builder

RUN apk add --no-cache git python3 py3-pip build-base
RUN corepack enable && corepack prepare pnpm@11.4.0 --activate

WORKDIR /build

# Copy the kiosk source from the host
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --node-linker=hoisted

COPY frontend/ ./
COPY shared/ ../shared/

RUN pnpm build

# ── Runtime stage ─────────────────────────────────────────────────────────
FROM node:${NODE_VERSION}-alpine

RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /build/.output /app/.output
COPY --from=builder /build/scripts/start-production.mjs /app/scripts/
COPY --from=builder /build/node_modules /app/node_modules
COPY --from=builder /build/package.json /app/

# Data directory for persistent JSON storage
RUN mkdir -p /app/data && chmod 777 /app/data

EXPOSE 3000 3001

ENV NODE_ENV=production
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000
ENV KIOSK_DATA_ROOT=/app/data

HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS http://localhost:3000/api/health || exit 1

CMD ["node", "scripts/start-production.mjs"]
