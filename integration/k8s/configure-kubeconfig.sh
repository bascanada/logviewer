#!/bin/bash
set -e

echo "Configuring kubectl for k3s..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KCFG="${SCRIPT_DIR}/k3s.yaml"
docker cp k3s-server:/etc/rancher/k3s/k3s.yaml "$KCFG"
sed -i -e "s/127.0.0.1/localhost/g" "$KCFG"
export KUBECONFIG="$KCFG"
echo "Kubeconfig written to $KCFG and environment variable KUBECONFIG set."
kubectl get nodes || echo "kubectl get nodes failed (cluster may still be starting)"
