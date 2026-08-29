#!/bin/sh
set -e

# Start Go backend server in background
/app/server &
SERVER_PID=$!

# Start Nginx in foreground
nginx -g "daemon off;" &
NGINX_PID=$!

# Trap signals for graceful shutdown
trap "kill -TERM $SERVER_PID $NGINX_PID; wait" SIGINT SIGTERM

wait
