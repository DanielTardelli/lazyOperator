#!/bin/sh

cd /workspaces/k8/lazy
make generate
make manifests
make docker-build IMG=lazy:latest
kind load docker-image lazy:latest --name testing-operator