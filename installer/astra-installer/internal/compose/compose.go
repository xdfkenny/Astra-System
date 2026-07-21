package compose

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Registry   string
	Tag        string
	KioskImage string
	DataDir    string
	KioskPort  string
	PostgresPW string
	JWTKey     string
}

func Generate(cfg Config, outputDir string) (string, error) {
	if cfg.JWTKey == "" {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return "", fmt.Errorf("generate jwt key pair: %w", err)
		}
		pubBytes, err := x509.MarshalPKIXPublicKey(pub)
		if err != nil {
			return "", fmt.Errorf("marshal public key: %w", err)
		}
		pemText := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubBytes,
		})
		cfg.JWTKey = base64.StdEncoding.EncodeToString(pemText)
	}
	content := buildCompose(cfg)
	outPath := filepath.Join(outputDir, "docker-compose.yml")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write compose file: %w", err)
	}
	return outPath, nil
}

func buildCompose(cfg Config) string {
	reg := cfg.Registry
	if reg == "" {
		reg = "ghcr.io/xdfkenny/astra-system"
	}
	tag := cfg.Tag
	if tag == "" {
		tag = "latest"
	}
	ki := cfg.KioskImage
	if ki == "" {
		ki = "kiosk"
	}
	img := func(n string) string { return fmt.Sprintf("%s/%s:%s", reg, n, tag) }

	return fmt.Sprintf(`services:
  postgres:
    image: postgres:16-alpine
    container_name: astra-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: astra
      POSTGRES_PASSWORD: %[1]s
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
    command: -js -m 8222 -cluster_name astra -server_name n1
    ports:
      - "4222:4222"
      - "8222:8222"

  gateway:
    image: %[2]s
    container_name: astra-gateway
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      REDIS_URL: redis:6379
      REDIS_ADDR: redis:6379
      NATS_URL: nats://nats:4222
      GATEWAY_JWT_EDDSA_PUBLIC_KEY: %[10]s
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  menu-service:
    image: %[3]s
    container_name: astra-menu
    restart: unless-stopped
    ports:
      - "8085:8085"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      NATS_URL: nats://nats:4222
      NATS_STREAM_REPLICAS: "1"
    depends_on:
      postgres:
        condition: service_healthy

  cart-service:
    image: %[4]s
    container_name: astra-cart
    restart: unless-stopped
    ports:
      - "8081:8081"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      CART_REDIS_ADDR: redis:6379
      NATS_URL: nats://nats:4222
      NATS_STREAM_REPLICAS: "1"
    depends_on:
      postgres:
        condition: service_healthy

  order-service:
    image: %[5]s
    container_name: astra-order
    restart: unless-stopped
    ports:
      - "8083:8083"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      NATS_URL: nats://nats:4222
      NATS_STREAM_REPLICAS: "1"
    depends_on:
      postgres:
        condition: service_healthy

  inventory-service:
    image: %[6]s
    container_name: astra-inventory
    restart: unless-stopped
    ports:
      - "8082:8082"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      NATS_URL: nats://nats:4222
      NATS_STREAM_REPLICAS: "1"
    depends_on:
      postgres:
        condition: service_healthy

  sync-service:
    image: %[7]s
    container_name: astra-sync
    restart: unless-stopped
    ports:
      - "8087:8087"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      NATS_URL: nats://nats:4222
      NATS_STREAM_REPLICAS: "1"
    depends_on:
      postgres:
        condition: service_healthy

  payment-orchestrator:
    image: %[8]s
    container_name: astra-payment
    restart: unless-stopped
    ports:
      - "8086:8086"
    environment:
      DATABASE_URL: postgres://astra:%[1]s@postgres:5432/astra_service?sslmode=disable
      REDIS_URL: redis://redis:6379/0
      NATS_URL: nats://nats:4222
      NATS_STREAM_REPLICAS: "1"
    depends_on:
      postgres:
        condition: service_healthy

  kiosk:
    image: %[9]s
    container_name: astra-kiosk
    restart: unless-stopped
    ports:
      - "80:80"
    tmpfs:
      - /var/cache/nginx:noexec,nosuid,size=50m
      - /var/run:noexec,nosuid,size=1m
    depends_on:
      gateway:
        condition: service_started

volumes:
  astra_postgres_data:
`, cfg.PostgresPW,
		img("gateway"),
		img("menu-service"),
		img("cart-service"),
		img("order-service"),
		img("inventory-service"),
		img("sync-service"),
		img("payment-orchestrator"),
		img(ki),
		cfg.JWTKey)
}

func sanitizeImageName(name string) string {
	return strings.ReplaceAll(strings.TrimSpace(name), " ", "-")
}
