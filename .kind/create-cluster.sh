#!/bin/sh

kind create cluster --name test-cluster --config /workspaces/k8/.kind/kind-config.yaml \
&& kubectl config set-context test-cluster

echo "Enter the ephemeral port mapped to the kind cluster container:"
read port

kubectl config set-cluster test-cluster --server https://host.docker.internal:$port

echo "Hooray"