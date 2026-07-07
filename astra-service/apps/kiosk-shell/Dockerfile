# syntax=docker/dockerfile:1.7

# Astra-Service Kiosk Shell (Simulator)
# Nginx-based static file serving for the production kiosk UI build

ARG NODE_VERSION=22

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS builder

RUN apk add --no-cache git ca-certificates
RUN corepack enable && corepack prepare pnpm@9.12.0 --activate

WORKDIR /build
COPY astra-service/package.json astra-service/pnpm-lock.yaml astra-service/pnpm-workspace.yaml astra-service/tsconfig.base.json astra-service/turbo.json ./
COPY astra-service/packages/ ./packages/
COPY astra-service/apps/kiosk-shell/ ./apps/kiosk-shell/

RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

RUN pnpm turbo run build --filter=@astra/kiosk-shell

FROM nginx:alpine
COPY --from=builder /build/apps/kiosk-shell/dist /usr/share/nginx/html
COPY infra/nginx/kiosk.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
STOPSIGNAL SIGTERM
