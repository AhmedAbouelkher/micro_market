#!/usr/bin/env bash
set -euo pipefail

cluster_name="${1:-micro-market}"

if ! kind get clusters | grep -qx "$cluster_name"; then
  kind create cluster --name "$cluster_name"
fi

echo "Building checkout-service:local"
docker build -t checkout-service:local -f checkout-service/Dockerfile . \
&& docker build -t inventory-service:local -f inventory-service/Dockerfile . \
&& docker build -t invoice-service:local -f invoice-service/Dockerfile . 

echo "Loading checkout-service:local into $cluster_name"
kind load docker-image checkout-service:local --name "$cluster_name" &
kind load docker-image inventory-service:local --name "$cluster_name" &
kind load docker-image invoice-service:local --name "$cluster_name" &
wait

kubectl apply -k k8s
