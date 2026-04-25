#!/usr/bin/env bash
set -euo pipefail

cluster_name="${1:-micro-market}"

if kind get clusters | grep -qx "$cluster_name"; then
  kind delete cluster --name "$cluster_name"
fi
