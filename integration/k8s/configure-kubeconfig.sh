#!/bin/bash
set -e

echo "Configuring kubectl for k3s..."
mkdir -p "$HOME/.kube"
docker cp k3s-server:/etc/rancher/k3s/k3s.yaml "$HOME/.kube/k3s.yaml"
sed -i -e "s/127.0.0.1/localhost/g" "$HOME/.kube/k3s.yaml"
export KUBECONFIG="$HOME/.kube/k3s.yaml"
echo "Kubeconfig is set. You can now use kubectl."
kubectl get nodes
