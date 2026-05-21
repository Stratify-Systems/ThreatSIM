#!/bin/bash
set -e

# ============================================================
# ThreatSIM CI/CD Security Validation Gate
# ============================================================
#
# This script validates that your security detection rules
# are working correctly by running a simulated attack and
# checking that alerts fire.
#
# Usage:
#   ./scripts/ci-validation.sh [plugin] [target] [rate] [duration]
#
# Exit codes:
#   0 = Detection validated (alerts fired) → safe to deploy
#   1 = Detection broken (no alerts) → block deployment
# ============================================================

PLUGIN=${1:-brute_force}
TARGET=${2:-http://localhost:9999/login}
RATE=${3:-15}
DURATION=${4:-5s}

# ThreatSIM API server address (separate from the target application)
THREATSIM_API="${THREATSIM_API:-http://localhost:8080/api/v1}"

echo "════════════════════════════════════════════════"
echo "🛡️  ThreatSIM Security Validation Gate"
echo "════════════════════════════════════════════════"
echo "  Target:    $TARGET"
echo "  Attack:    $PLUGIN"
echo "  Rate:      $RATE events/sec"
echo "  Duration:  $DURATION"
echo "  API:       $THREATSIM_API"
echo "════════════════════════════════════════════════"
echo ""

# ── Step 1: Health check ──────────────────────────
echo "⏳ [Step 1/4] Checking ThreatSIM API health..."
MAX_RETRIES=10
for i in $(seq 1 $MAX_RETRIES); do
    if curl -s -f "${THREATSIM_API%/api/v1}/health" > /dev/null 2>&1; then
        echo "✅ API is healthy."
        break
    fi
    if [ "$i" -eq "$MAX_RETRIES" ]; then
        echo "❌ ThreatSIM API not reachable after $MAX_RETRIES attempts."
        exit 1
    fi
    echo "  Retrying in 2s... ($i/$MAX_RETRIES)"
    sleep 2
done

# ── Step 2: Trigger the attack simulation ─────────
echo ""
echo "🚀 [Step 2/4] Triggering $PLUGIN simulation..."
RESPONSE=$(curl -s -X POST "$THREATSIM_API/simulations" \
    -H "Content-Type: application/json" \
    -d "{\"plugin_id\": \"$PLUGIN\", \"target\": \"$TARGET\", \"rate\": $RATE, \"duration\": \"$DURATION\"}")

SIM_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ -z "$SIM_ID" ]; then
    echo "❌ Failed to start simulation. Response:"
    echo "   $RESPONSE"
    exit 1
fi

echo "✅ Simulation started (ID: $SIM_ID)"

# ── Step 3: Wait for detection processing ─────────
# The detection engine needs time to:
# 1. Receive all events from the simulation
# 2. Evaluate sliding window thresholds
# 3. Generate risk scores

# Parse duration to seconds (simple: strip the 's' suffix)
WAIT_SECS=$(echo "$DURATION" | sed 's/s$//')
WAIT_TIME=$((WAIT_SECS + 5))

echo ""
echo "⏳ [Step 3/4] Waiting ${WAIT_TIME}s for detection processing..."
sleep "$WAIT_TIME"

# ── Step 4: Check for alerts ──────────────────────
echo ""
echo "📊 [Step 4/4] Querying alert engine for detections..."
ALERTS=$(curl -s "$THREATSIM_API/alerts")

# Check if HIGH or CRITICAL alerts were generated
if echo "$ALERTS" | grep -q '"threat_level":"CRITICAL"' || echo "$ALERTS" | grep -q '"threat_level":"HIGH"'; then
    echo ""
    echo "════════════════════════════════════════════════"
    echo "✅ VALIDATION PASSED"
    echo "════════════════════════════════════════════════"
    echo "  Detection engine caught the $PLUGIN attack."
    echo "  Alert data:"
    echo "$ALERTS" | grep -o '"threat_level":"[^"]*"' | head -n 3
    echo ""
    echo "  ➡️  Safe to deploy to production."
    echo "════════════════════════════════════════════════"
    exit 0
else
    echo ""
    echo "════════════════════════════════════════════════"
    echo "❌ VALIDATION FAILED"
    echo "════════════════════════════════════════════════"
    echo "  No HIGH/CRITICAL alerts detected."
    echo "  Your detection rules may be broken or logging has failed."
    echo ""
    echo "  ➡️  Deployment BLOCKED."
    echo "════════════════════════════════════════════════"
    exit 1
fi
