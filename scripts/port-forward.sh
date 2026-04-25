#!/usr/bin/env bash
set -euo pipefail

ns="${1:-micro-market}"

cleanup() {
  jobs -p | xargs -r kill
}

trap cleanup INT TERM EXIT

kubectl -n "$ns" port-forward svc/checkout 8888:1234 >/tmp/checkout.pf.log 2>&1 &
kubectl -n "$ns" port-forward svc/inventory 9999:1234 >/tmp/inventory.pf.log 2>&1 &
kubectl -n "$ns" port-forward svc/grafana-otel 3000:3000 >/tmp/grafana.pf.log 2>&1 &

wait
