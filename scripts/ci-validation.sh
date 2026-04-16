#!/bin/bash
set -e

# ThreatSIM Security validation CI/CD script
# Validates that a detection rule triggers as expected when ThreatSIM attacks a target.

PLUGIN=${1:-brute_force}
TARGET=${2:-http://localhost:8080}
RATE=${3:-10}
DURATION=${4:-5s}

THREATSIM_API="http://localhost:8080/api/v1"

echo "--------------------------------------------------------"
echo "🔍 Starting Security CI/CD Validation Gate"
echo "👉 Target: $TARGET"
echo "👉 Attack Plugin: $PLUGIN"
echo "--------------------------------------------------------"

# 1. Execute the simulated attack
# In a real environment, you might use the fully built CLI tool (./threatsim simulate ...).
# For CI automation, we trigger the simulation directly against the ThreatSIM engine API.

echo "🚀 [Step 1] Triggering Attack Simulation..."
RESPONSE=$(curl -s -X POST $THREATSIM_API/simulations \
    -H "Content-Type: application/json" \
    -d "{\"plugin_id\": \"$PLUGIN\", \"target\": \"$TARGET\", \"rate\": $RATE, \"duration\": \"$DURATION\"}")

SIM_ID=$(echo $RESPONSE | grep -o 'id":"[^"]*' | cut -d'"' -f3)

if [ -z "$SIM_ID" ]; then
    echo "❌ [ERROR] Failed to start simulation. Check ThreatSIM backend."
    echo $RESPONSE
    exit 1
fi

echo "✅ Simulation started successfully (ID: $SIM_ID)"

# Wait for 10 seconds to allow the backend Detection & Risk engine to process events and issue alerts.
WAIT_TIME=10
echo "⏳ [Step 2] Waiting ${WAIT_TIME}s for target logs and detection rules to process..."
sleep $WAIT_TIME

# 2. Check for alerts
# We query the SIEM/Alert API to verify if the attacker IP was flagged.

echo "📊 [Step 3] Querying SIEM / Alert Engine for Detections..."
ALERTS=$(curl -s $THREATSIM_API/alerts)

# Use basic grep to detect if the target was flagged
# If this was a valid attack, the 'threat_level' or 'score' should be visible.
if echo "$ALERTS" | grep -q "\"threat_level\":\"CRITICAL\"" || echo "$ALERTS" | grep -q "\"threat_level\":\"HIGH\""; then
    echo "✅ [Step 4] Validation Passed! Detection Engine caught the attack."
    echo "   Alert data found:"
    echo "$ALERTS" | grep -o '"threat_level":"[^"]*"' | head -n 1
    echo "   Pipeline may proceed -> Deploy to Production."
    exit 0
else
    echo "❌ [Step 4] Validation Failed! No matching high/critical alerts discovered."
    echo "   Your detection rules might be broken, or logging has failed."
    echo "   Pipeline blocked -> Halting Deployment."
    exit 1
fi
