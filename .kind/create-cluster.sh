#!/bin/sh

kind create cluster --name test-cluster --config /workspaces/k8/.kind/kind-config.yaml \
&& kubectl config set-context test-cluster \
&& kubectl config set-cluster test-cluster --server https://host.docker.internal:45355

echo "Hooray"