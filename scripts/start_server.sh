#!/bin/bash

# Get the absolute project root directory
PROJECT_ROOT="$(cd "$(dirname "$0")/.."; pwd)"
cd "$PROJECT_ROOT"

# Kill any existing process on port 8080
echo "Checking for existing process on port 8080..."
lsof -ti:8080 | xargs kill -9 2>/dev/null || true

# Build the server
echo "Building the server..."
cd "$PROJECT_ROOT/backend"
go build -o summarize-server main.go

# Load environment variables from local.env
set -a
source "$PROJECT_ROOT/local.env"
set +a

# Start the Go backend in the background
./summarize-server &
BACKEND_PID=$!
echo "Started Go backend with PID $BACKEND_PID"

# Wait a moment to ensure the backend is running
sleep 2

# Start ngrok in the background
echo "Starting ngrok tunnel on port 8080..."
ngrok http 8080 --log=stdout > "$PROJECT_ROOT/ngrok.log" &
NGROK_PID=$!

# Wait for ngrok to initialize and get the URL using jq
for i in {1..10}; do
  NGROK_URL=$(curl --silent http://localhost:4040/api/tunnels | jq -r '.tunnels[] | select(.proto == "https") | .public_url' | head -n 1)
  if [[ $NGROK_URL == https://* ]]; then
    break
  fi
  sleep 1
done
echo "ngrok public URL: $NGROK_URL"

# Update the fetch URL in popup.js with the new ngrok URL
cd "$PROJECT_ROOT"
sed -i '' "s|https://[^/]*\.ngrok-free\.app|$NGROK_URL|g" popup.js

export NGROK_HOST=$NGROK_URL

echo "Updated line 32 of popup.js to use ngrok URL: $NGROK_URL"

# Wait for background processes
wait $BACKEND_PID $NGROK_PID