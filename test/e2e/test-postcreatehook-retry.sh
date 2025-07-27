#!/usr/bin/env bash

set -e

echo "🧹 Cleaning up any existing resources..."
kubectl delete controlplane cp-missing-hook --ignore-not-found=true
kubectl delete postcreatehook missing-hook --ignore-not-found=true

echo ""
echo "🔧 Creating ControlPlane referencing a missing PostCreateHook..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp-missing-hook
spec:
  backend: shared
  postCreateHook: missing-hook
  waitForPostCreateHooks: true
  type: k8s
EOF

echo ""
echo "⏳ Waiting 10s to check that ControlPlane is not marked as failed..."
sleep 10

echo ""
echo "📋 ControlPlane status after 10s (should NOT be failed):"
kubectl get controlplane cp-missing-hook -o jsonpath='{.status.conditions}' | jq '.'

echo ""
echo "🧪 Creating the missing PostCreateHook..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: missing-hook
spec:
  templates:
  - apiVersion: batch/v1
    kind: Job
    metadata:
      name: job-missing-hook
    spec:
      template:
        spec:
          containers:
          - name: demo
            image: public.ecr.aws/docker/library/busybox:1.36
            command: ["echo", "Hello from missing hook"]
          restartPolicy: Never
      backoffLimit: 1
EOF

echo ""
echo "⏳ Waiting for ControlPlane to become Ready (90s timeout)..."
kubectl wait --for=condition=Ready controlplane/cp-missing-hook --timeout=90s

echo ""
echo "📊 FINAL STATUS:"
kubectl get controlplane cp-missing-hook -o jsonpath='{.status}' | jq '.'

echo ""
echo "🧹 Cleaning up test resources..."
kubectl delete controlplane cp-missing-hook --ignore-not-found=true
kubectl delete postcreatehook missing-hook --ignore-not-found=true