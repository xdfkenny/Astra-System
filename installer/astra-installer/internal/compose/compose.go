package compose

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Registry   string
	Tag        string
	KioskImage string
	DataDir    string
	KioskPort  string
	PostgresPW string
}

func Generate(cfg Config, outputDir string) (string, error) {
	content := buildCompose(cfg)
	outPath := filepath.Join(outputDir, "docker-compose.yml")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write compose file: %w", err)
	}
	return outPath, nil
}

func buildCompose(cfg Config) string {
	registry := cfg.Registry
	if registry == "" {
		registry = "ghcr.io/xdfkenny/astra-system"
	}
	tag := cfg.Tag
	if tag == "" {
		tag = "latest"
	}
	pImg := func(name string) string {
		return fmt.Sprintf("%s/%s:%s", registry, name, tag)
	}

	return fmt.Sprintf(`services:
  postgres:
    image: postgres:16-alpine
    container_name: astra-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: astra
      POSTGRES_PASSWORD: %s
      POSTGRES_DB: astra_service
    ports:
      - "5432:5432"
    volumes:
      - astra_postgres_data:/var/lib/postgresql/data
      - ./init:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U astra -d astra_service"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: astra-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2-alpine
    container_name: astra-nats
    restart: unless-stopped
    ports:
      - "4222:4222"
      - "8222:8222"
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8222/healthz || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  gateway:
    image: %s
    container_name: astra-gateway
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      REDIS_ADDR: redis:6379
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }
      nats: { condition: service_healthy }

  menu-service:
    image: %s
    container_name: astra-menu
    restart: unless-stopped
    ports:
      - "8085:8085"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  cart-service:
    image: %s
    container_name: astra-cart
    restart: unless-stopped
    ports:
      - "8081:8081"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  order-service:
    image: %s
    container_name: astra-order
    restart: unless-stopped
    ports:
      - "8083:8083"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  inventory-service:
    image: %s
    container_name: astra-inventory
    restart: unless-stopped
    ports:
      - "8082:8082"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  sync-service:
    image: %s
    container_name: astra-sync
    restart: unless-stopped
    ports:
      - "8087:8087"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  payment-orchestrator:
    image: %s
    container_name: astra-payment
    restart: unless-stopped
    ports:
      - "8086:8086"
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: astra
      DB_PASSWORD: %s
      DB_NAME: astra_service
      NATS_URL: nats:4222
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_healthy }

  kiosk:
    image: %s
    container_name: astra-kiosk
    restart: unless-stopped
    ports:
      - "%s:80"
    depends_on:
      - gateway

volumes:
  astra_postgres_data:
`, cfg.PostgresPW,
		pImg("gateway"), cfg.PostgresPW,
		pImg("menu-service"), cfg.PostgresPW,
		pImg("cart-service"), cfg.PostgresPW,
		pImg("order-service"), cfg.PostgresPW,
		pImg("inventory-service"), cfg.PostgresPW,
		pImg("sync-service"), cfg.PostgresPW,
		pImg("payment-orchestrator"), cfg.PostgresPW,
		pImg(cfg.KioskImage), cfg.KioskPort)
}


