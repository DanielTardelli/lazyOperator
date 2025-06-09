#!/bin/sh

echo "port forwarding argo service"
kubectl port-forward service/argocd-server -n argocd 2746:443 &
PF_PID=$!

echo "redirecting all traffic 2747 to localhost 2746 for container to get around k8 port forwarding restrictions"
socat TCP-LISTEN:2747,fork,reuseaddr TCP:127.0.0.1:2746 &
SOCAT_PID=$!

cleanup() {
  echo "Stopping port-forward and socat..."
  kill $PF_PID $SOCAT_PID
  exit 0
}

trap cleanup INT TERM

# Wait forever (or do something else)
wait $PF_PID $SOCAT_PID
