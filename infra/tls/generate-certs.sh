#!/usr/bin/env bash
# Astra-Service PKI generation script.
#
# Generates a complete internal PKI for local development and CI:
#   - Root CA
#   - Server certificates for every service (with SANs for localhost + service names)
#   - Client certificate for mTLS client authentication
#   - Kiosk device certificate
#
# Requires either cfssl/cfssljson (preferred) or openssl (fallback).
# All generated artifacts are written to ./out and MUST NOT be committed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${SCRIPT_DIR}/out"
CONFIG_DIR="${SCRIPT_DIR}/config"

cd "${SCRIPT_DIR}"

mkdir -p "${OUT_DIR}" "${CONFIG_DIR}"

# Certificate validity in hours (10 years for dev/CI)
VALIDITY_HOURS=87600

# Service list: name port
SERVICES=(
  "gateway 8080"
  "cart-service 8081"
  "inventory-service 8082"
  "order-service 8083"
  "payment-service 8084"
  "menu-service 8085"
  "payment-orchestrator 8086"
  "sync-service 8087"
  "update-server 8090"
  "ml-lane-intel 8088"
  "kiosk 80"
  "sync-daemon 4499"
  "postgres 5432"
  "redis 6379"
  "nats 4222"
)

echo "== Astra-Service PKI generator =="

# Detect available tooling
USE_CFSSL=false
if command -v cfssl >/dev/null 2>&1 && command -v cfssljson >/dev/null 2>&1; then
  USE_CFSSL=true
  echo "Using cfssl/cfssljson for certificate generation."
else
  echo "cfssl not found; falling back to openssl."
fi

# ---------------------------------------------------------------------------
# cfssl path
# ---------------------------------------------------------------------------
generate_with_cfssl() {
  cat > "${CONFIG_DIR}/ca-csr.json" <<'EOF'
{
  "CN": "Astra-Service Internal CA",
  "key": { "algo": "ecdsa", "size": 256 },
  "names": [
    { "C": "US", "O": "Astra Systems", "OU": "Platform Security" }
  ],
  "ca": { "expiry": "87600h" }
}
EOF

  cat > "${CONFIG_DIR}/ca-config.json" <<'EOF'
{
  "signing": {
    "default": { "expiry": "87600h" },
    "profiles": {
      "server": {
        "expiry": "87600h",
        "usages": ["signing", "key encipherment", "server auth"]
      },
      "client": {
        "expiry": "87600h",
        "usages": ["signing", "key encipherment", "client auth"]
      },
      "peer": {
        "expiry": "87600h",
        "usages": ["signing", "key encipherment", "server auth", "client auth"]
      }
    }
  }
}
EOF

  cfssl gencert -initca "${CONFIG_DIR}/ca-csr.json" | cfssljson -bare "${OUT_DIR}/ca"

  for entry in "${SERVICES[@]}"; do
    svc="${entry% *}"
    port="${entry#* }"
    san_hosts="\"localhost\",\"${svc}\",\"${svc}.astra-net\",\"${svc}.astra-service.svc.cluster.local\""
    if [[ "$svc" == "gateway" ]]; then
      san_hosts="\"localhost\",\"gateway\",\"gateway.astra-net\",\"api.astra.local\",\"*.astra.local\""
    fi
    if [[ "$svc" == "ml-lane-intel" ]]; then
      san_hosts="\"localhost\",\"ml-lane-intel\",\"ml-lane-intel.astra-net\""
    fi

    cat > "${CONFIG_DIR}/${svc}-csr.json" <<EOF
{
  "CN": "${svc}.astra-service",
  "hosts": [${san_hosts}],
  "key": { "algo": "ecdsa", "size": 256 },
  "names": [
    { "C": "US", "O": "Astra Systems", "OU": "${svc}" }
  ]
}
EOF

    cfssl gencert \
      -ca="${OUT_DIR}/ca.pem" \
      -ca-key="${OUT_DIR}/ca-key.pem" \
      -config="${CONFIG_DIR}/ca-config.json" \
      -profile=server \
      "${CONFIG_DIR}/${svc}-csr.json" | cfssljson -bare "${OUT_DIR}/${svc}"
  done

  # Client certificate for service-to-service mTLS
  cat > "${CONFIG_DIR}/client-csr.json" <<'EOF'
{
  "CN": "astra-service-client",
  "hosts": [""],
  "key": { "algo": "ecdsa", "size": 256 },
  "names": [
    { "C": "US", "O": "Astra Systems", "OU": "service-client" }
  ]
}
EOF
  cfssl gencert \
    -ca="${OUT_DIR}/ca.pem" \
    -ca-key="${OUT_DIR}/ca-key.pem" \
    -config="${CONFIG_DIR}/ca-config.json" \
    -profile=client \
    "${CONFIG_DIR}/client-csr.json" | cfssljson -bare "${OUT_DIR}/client"

  # Kiosk device certificate (peer profile)
  cat > "${CONFIG_DIR}/kiosk-csr.json" <<'EOF'
{
  "CN": "kiosk-sim-001.astra-service",
  "hosts": ["localhost", "kiosk-sim-001", "kiosk"],
  "key": { "algo": "ecdsa", "size": 256 },
  "names": [
    { "C": "US", "O": "Astra Systems", "OU": "kiosk" }
  ]
}
EOF
  cfssl gencert \
    -ca="${OUT_DIR}/ca.pem" \
    -ca-key="${OUT_DIR}/ca-key.pem" \
    -config="${CONFIG_DIR}/ca-config.json" \
    -profile=peer \
    "${CONFIG_DIR}/kiosk-csr.json" | cfssljson -bare "${OUT_DIR}/kiosk"
}

# ---------------------------------------------------------------------------
# openssl fallback path
# ---------------------------------------------------------------------------
generate_with_openssl() {
  : "${OPENSSL:=openssl}"

  # Root CA key + cert
  "${OPENSSL}" genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "${OUT_DIR}/ca-key.pem"
  "${OPENSSL}" req -x509 -new -key "${OUT_DIR}/ca-key.pem" -sha256 -days 3650 \
    -subj "/C=US/O=Astra Systems/OU=Platform Security/CN=Astra-Service Internal CA" \
    -out "${OUT_DIR}/ca.pem"

  for entry in "${SERVICES[@]}"; do
    svc="${entry% *}"
    port="${entry#* }"
    san_hosts="DNS:localhost,DNS:${svc},DNS:${svc}.astra-net,DNS:${svc}.astra-service.svc.cluster.local"
    if [[ "$svc" == "gateway" ]]; then
      san_hosts="DNS:localhost,DNS:gateway,DNS:gateway.astra-net,DNS:api.astra.local,DNS:*.astra.local"
    fi
    if [[ "$svc" == "ml-lane-intel" ]]; then
      san_hosts="DNS:localhost,DNS:ml-lane-intel,DNS:ml-lane-intel.astra-net"
    fi

    cat > "${CONFIG_DIR}/${svc}.ext" <<EOF
basicConstraints=CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = ${san_hosts}
EOF

    "${OPENSSL}" genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "${OUT_DIR}/${svc}-key.pem"
    "${OPENSSL}" req -new -key "${OUT_DIR}/${svc}-key.pem" \
      -subj "/C=US/O=Astra Systems/OU=${svc}/CN=${svc}.astra-service" \
      -out "${CONFIG_DIR}/${svc}.csr"
    "${OPENSSL}" x509 -req -in "${CONFIG_DIR}/${svc}.csr" -CA "${OUT_DIR}/ca.pem" -CAkey "${OUT_DIR}/ca-key.pem" \
      -CAcreateserial -out "${OUT_DIR}/${svc}.pem" -days 3650 -sha256 -extfile "${CONFIG_DIR}/${svc}.ext"
  done

  # Client cert
  cat > "${CONFIG_DIR}/client.ext" <<'EOF'
basicConstraints=CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF
  "${OPENSSL}" genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "${OUT_DIR}/client-key.pem"
  "${OPENSSL}" req -new -key "${OUT_DIR}/client-key.pem" \
    -subj "/C=US/O=Astra Systems/OU=service-client/CN=astra-service-client" \
    -out "${CONFIG_DIR}/client.csr"
  "${OPENSSL}" x509 -req -in "${CONFIG_DIR}/client.csr" -CA "${OUT_DIR}/ca.pem" -CAkey "${OUT_DIR}/ca-key.pem" \
    -CAcreateserial -out "${OUT_DIR}/client.pem" -days 3650 -sha256 -extfile "${CONFIG_DIR}/client.ext"

  # Kiosk device cert (server + client auth)
  cat > "${CONFIG_DIR}/kiosk.ext" <<'EOF'
basicConstraints=CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = DNS:localhost,DNS:kiosk-sim-001,DNS:kiosk
EOF
  "${OPENSSL}" genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out "${OUT_DIR}/kiosk-key.pem"
  "${OPENSSL}" req -new -key "${OUT_DIR}/kiosk-key.pem" \
    -subj "/C=US/O=Astra Systems/OU=kiosk/CN=kiosk-sim-001.astra-service" \
    -out "${CONFIG_DIR}/kiosk.csr"
  "${OPENSSL}" x509 -req -in "${CONFIG_DIR}/kiosk.csr" -CA "${OUT_DIR}/ca.pem" -CAkey "${OUT_DIR}/ca-key.pem" \
    -CAcreateserial -out "${OUT_DIR}/kiosk.pem" -days 3650 -sha256 -extfile "${CONFIG_DIR}/kiosk.ext"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
if [[ "${USE_CFSSL}" == "true" ]]; then
  generate_with_cfssl
else
  generate_with_openssl
fi

# Normalize permissions
chmod 644 "${OUT_DIR}"/*.pem "${OUT_DIR}"/*.csr 2>/dev/null || true
chmod 600 "${OUT_DIR}"/*-key.pem

echo ""
echo "PKI generated in ${OUT_DIR}:"
ls -1 "${OUT_DIR}"
echo ""
echo "Next steps:"
echo "  1. Mount ${OUT_DIR}/ca.pem as a trusted CA in each container."
echo "  2. Mount service keypair into /etc/astra/certs for each service."
echo "  3. NEVER commit ${OUT_DIR} to version control."
