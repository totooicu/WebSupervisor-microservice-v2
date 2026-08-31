#!/bin/bash
# Launch all 5 services in the background, each writing to its own log file.
# Logs land in bin/<service>/<service>.log
# Requires: Redis reachable per each config.json.
# Tip:  tail -f */*.log        # follow all logs
#       pkill -f '_linux_'     # stop all services

cd "$(dirname "$0")" || exit 1

services=(crawler-service email-service parser-service redis_cache-service web_supervisor-manager)

echo "=== Launching all services ==="
for s in "${services[@]}"; do
    if [ -x "$s/run_linux.sh" ] || [ -f "$s/run_linux.sh" ]; then
        echo "  starting $s ... (log: $s/$s.log)"
        ( cd "$s" && ./run_linux.sh > "$s.log" 2>&1 & )
    else
        echo "  [SKIP] $s/run_linux.sh not found"
    fi
done
echo
echo "All services launched in background."
echo "Follow logs:  tail -f */*.log"
echo "Stop all:     pkill -f '_linux_'"
