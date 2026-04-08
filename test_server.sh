pkill threatsim || true
sleep 1
./bin/threatsim server > server.log 2>&1 &
SERVER_PID=$!
sleep 2

echo "== Server Startup Logs =="
cat server.log
echo ""

echo "== Triggering Account Takeover Scenario =="
curl -s -X POST http://localhost:8080/api/v1/scenarios -H "Content-Type: application/json" -d '{"scenario_id": "account_takeover", "target": "127.0.0.1"}'
echo -e "\nWaiting 5 seconds for scenario completion..."
sleep 5

echo "== Fetching Events (showing first lines) =="
curl -s http://localhost:8080/api/v1/events | jq '.[0:3]' || true

echo "== Fetching Simulations (verifying scenario metadata logged) =="
curl -s http://localhost:8080/api/v1/simulations | jq '.' | grep -i scenario

kill $SERVER_PID || true
echo "== Test complete! =="
