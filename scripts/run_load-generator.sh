#!/bin/zsh

export CHECKOUT_URL=http://localhost:8888
export INVENTORY_URL=http://localhost:9999
export DURATION=30s
export INTERVAL=500ms
export SEED_PRODUCTS=5
export USER_ID=1

if ! go run cmd/load-generator/main.go; then
  if ! go run main.go; then
    echo "Failed to run load generator: could not run via either cmd/load-generator/main.go or main.go" >&2
    exit 1
  fi
fi