#!/usr/bin/env bash

echo "🧪 Creating PostCreateHook..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: demo-hook
spec:
  templates:
  - apiVersion: batch/v1
    kind: Job
    metadata:
      name: demo-job-{{.ControlPlaneName}}
    spec:
      template:
        spec:
          containers:
          - name: demo
            image: public.ecr.aws/docker/library/busybox:1.36
            command: ["sleep", "15"]
          restartPolicy: Never
      backoffLimit: 1
EOF

echo ""
echo "🔧 Creating CP with waitForPostCreateHooks=TRUE..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp-wait-true
spec:
  backend: shared
  postCreateHook: demo-hook
  waitForPostCreateHooks: true
  type: k8s
EOF

echo ""
echo "⚡ Creating CP with waitForPostCreateHooks=FALSE..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp-wait-false
spec:
  backend: shared
  postCreateHook: demo-hook
  waitForPostCreateHooks: false
  type: k8s
EOF

echo ""
echo "⏳ Waiting for both CPs to be ready (90s timeout)..."
kubectl wait --for=condition=Ready controlplane/cp-wait-true --timeout=90s &  #set time accoeding to you!
kubectl wait --for=condition=Ready controlplane/cp-wait-false --timeout=90s &
wait

echo ""
echo "📊 RESULTS:"
echo ""
echo "=== CP with waitForPostCreateHooks=TRUE ==="
kubectl get controlplane cp-wait-true -o jsonpath='{.status}' | jq '.'

echo ""
echo "=== CP with waitForPostCreateHooks=FALSE ==="
kubectl get controlplane cp-wait-false -o jsonpath='{.status}' | jq '.'

echo ""
echo "📋 Summary:"
kubectl get cp cp-wait-true cp-wait-false

#please remove this lines so that after test all resurce will be deleted!
# echo "🧹 Cleaning up any existing resources..."
# kubectl delete controlplane cp-wait-true --ignore-not-found=true
# kubectl delete controlplane cp-wait-false --ignore-not-found=true
# kubectl delete postcreatehook demo-hook --ignore-not-found=true