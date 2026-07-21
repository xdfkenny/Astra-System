#!/usr/bin/env bash
# Smoke test for Astra-System installer
# Run: bash installer/scripts/smoke-test.sh
set -uo pipefail
cd "$(dirname "$0")/../.."

PASS=0
FAIL=0

pass() { PASS=$((PASS+1)); printf "  ✓ %s\n" "$1"; }
fail() { FAIL=$((FAIL+1)); printf "  ✗ %s\n" "$1"; }
check() {
  local label="$1" expected="$2"; shift 2
  local output
  output=$("$@" 2>&1) || true
  if echo "$output" | grep -qc "$expected"; then
    pass "$label"
  else
    fail "$label (expected: '$expected', got: '$output')"
  fi
}

BIN="/tmp/astra-installer-test"
rm -f "$BIN"
(cd installer/astra-installer && go build -o "$BIN" ./cmd/astra-installer)

echo "╔═══════════════════════════════════════════╗"
echo "║  Astra-System Installer Smoke Tests       ║"
echo "╚═══════════════════════════════════════════╝"
echo ""

# ── CLI validation ──
echo "── CLI flags ──"
check "--version"       "v0.2.0"              "$BIN" --version
check "bad port (0)"    "invalid.*kiosk"      "$BIN" --kiosk-port 0 --silent
check "bad port (abc)"  "invalid.*kiosk"      "$BIN" --kiosk-port abc --silent
check "bad port (70k)"  "invalid.*kiosk"      "$BIN" --kiosk-port 70000 --silent
check "bad channel"     "invalid.*channel"    "$BIN" --channel nightly --silent
check "empty tag"       "must not be empty"   "$BIN" --tag "" --silent
check "empty registry"  "must not be empty"   "$BIN" --registry "" --silent

# ── Remove ──
echo "── Uninstall ──"
check "--remove (non-root)" "Astra-System removed" "$BIN" --remove --install-dir /tmp/atest --data-dir /tmp/atest-data

# ── Docker check ──
echo "── Docker ──"
check "docker not running" "Docker daemon not running" "$BIN" --silent --install-dir /tmp/atest --data-dir /tmp/atest-data

# ── Code quality ──
echo "── Code quality ──"
(cd installer/astra-installer && go vet ./...) 2>&1 && pass "go vet" || fail "go vet"

# ── Cross-compile ──
echo "── Cross-compile ──"
for target in "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64"; do
  GOOS="${target%/*}" GOARCH="${target#*/}"
  if (cd installer/astra-installer && CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -o /dev/null ./cmd/astra-installer) 2>/dev/null; then
    pass "$target"
  else
    fail "$target"
  fi
done

# ── Scripts ──
echo "── Scripts ──"
if command -v bash &>/dev/null; then
  bash -n installer/scripts/install.sh && pass "install.sh syntax" || fail "install.sh syntax"
fi

# ── Summary ──
echo ""
echo "╔═══════════════════════════════════════════╗"
printf "║  %d passed, %d failed                      ║\n" $PASS $FAIL
echo "╚═══════════════════════════════════════════╝"

[ "$FAIL" -eq 0 ]
