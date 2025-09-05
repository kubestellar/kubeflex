#!/usr/bin/env bash

set -x # echo commands for better debugging
set -e # exit on error

CP_TYPE="k8s"
DEBUG=false

while (( $# > 0 )); do
  case "$1" in
  (-t|--type)
    if (( $# > 1 ));
    then { CP_TYPE="$2"; shift 2; }
    else { echo "missing value for controlplane type" >&2; exit 1; }
    fi;;
  (-d|--debug)
    DEBUG=true
    shift;;
  (-*)
    echo "unknown flag: $1" >&2
    exit 1;;
  (*)
    echo "unknown positional argument: $1" >&2
    exit 1;;
  esac
done

echo "🧪 Testing PostCreateHook completion behavior with ${CP_TYPE} control plane..."

echo ""
echo "🧹 Cleaning up any existing resources..."
kubectl delete controlplane cp-wait-true-${CP_TYPE} --ignore-not-found=true
kubectl delete controlplane cp-wait-false-${CP_TYPE} --ignore-not-found=true
kubectl delete postcreatehook demo-hook-${CP_TYPE} --ignore-not-found=true

echo ""
echo "🔨 Creating PostCreateHook (${CP_TYPE})..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: demo-hook-${CP_TYPE}
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
echo "🔧 Creating CP with waitForPostCreateHooks=TRUE (${CP_TYPE})..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp-wait-true-${CP_TYPE}
spec:
  backend: shared
  postCreateHook: demo-hook-${CP_TYPE}
  waitForPostCreateHooks: true
  type: ${CP_TYPE}
EOF

echo ""
echo "⚡ Creating CP with waitForPostCreateHooks=FALSE (${CP_TYPE})..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp-wait-false-${CP_TYPE}
spec:
  backend: shared
  postCreateHook: demo-hook-${CP_TYPE}
  waitForPostCreateHooks: false
  type: ${CP_TYPE}
EOF

echo ""
echo "⏳ Waiting for ${CP_TYPE} CP to be ready..."
kubectl wait --for=condition=Ready controlplane/cp-wait-true-${CP_TYPE} --timeout=600s &
kubectl wait --for=condition=Ready controlplane/cp-wait-false-${CP_TYPE} --timeout=600s &
wait

echo ""
echo "📊 RESULTS for ${CP_TYPE} CP:"
echo ""
echo "=== CP with waitForPostCreateHooks=TRUE ==="
kubectl get controlplane cp-wait-true-${CP_TYPE} -o jsonpath='{.status}' | jq '.'

echo ""
echo "=== CP with waitForPostCreateHooks=FALSE ==="
kubectl get controlplane cp-wait-false-${CP_TYPE} -o jsonpath='{.status}' | jq '.'

echo ""
echo "📋 Summary:"
kubectl get cp cp-wait-true-${CP_TYPE} cp-wait-false-${CP_TYPE}

if [[ "$DEBUG" != "true" ]]; then
  echo ""
  echo "🧹 Cleaning up any existing resources..."
  kubectl delete controlplane cp-wait-true-${CP_TYPE} --ignore-not-found=true
  kubectl delete controlplane cp-wait-false-${CP_TYPE} --ignore-not-found=true
  kubectl delete postcreatehook demo-hook-${CP_TYPE} --ignore-not-found=true
fi

echo "" 
echo "✅ SUCCESS: ${CP_TYPE} PostCreateHook completion test completed"
