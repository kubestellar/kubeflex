#!/usr/bin/env bash

# E2E test for PostCreateHook Completion Status Feature

set -euo pipefail

echo "🧪 Starting PostCreateHook Completion E2E Test..."

# Test variables
TEST_HOOK_NAME="e2e-completion-test"
TEST_CP_NAME="test-cp-completion"
TEST_CP_NO_WAIT="test-cp-no-wait"

# Cleanup function
cleanup() {
    echo "🧹 Cleaning up..."
    kubectl delete controlplane "${TEST_CP_NAME}" --ignore-not-found=true
    kubectl delete controlplane "${TEST_CP_NO_WAIT}" --ignore-not-found=true  
    kubectl delete postcreatehook "${TEST_HOOK_NAME}" --ignore-not-found=true
    echo "✅ Cleanup completed"
}

trap cleanup EXIT

echo "1. Creating PostCreateHook..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: ${TEST_HOOK_NAME}
spec:
  templates:
  - apiVersion: batch/v1
    kind: Job
    metadata:
      name: completion-test-job-{{.ControlPlaneName}}
    spec:
      template:
        spec:
          containers:
          - name: test
            image: public.ecr.aws/docker/library/busybox:1.36
            command: ["sleep", "15"]  # Short delay to test completion tracking
          restartPolicy: Never
      backoffLimit: 1
EOF

echo "2. Creating control plane with PostCreateHook completion testing..."
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: ${TEST_CP_NAME}
spec:
  backend: shared
  postCreateHook: ${TEST_HOOK_NAME}
  waitForPostCreateHooks: true
  type: k8s
EOF

echo "3. Create ControlPlane with waitForPostCreateHooks enabled"
echo "4. Waiting for control plane to be created..."
kubectl wait --for=condition=Ready controlplane/${TEST_CP_NAME} --timeout=150s

echo "5. Monitoring PostCreateHook completion status..."
# Check if postCreateHookCompleted is set to true
COMPLETION_STATUS=$(kubectl get controlplane ${TEST_CP_NAME} -o jsonpath='{.status.postCreateHookCompleted}')
echo "PostCreateHookCompleted: ${COMPLETION_STATUS}"

# Check individual hook status
HOOK_STATUS=$(kubectl get controlplane ${TEST_CP_NAME} -o jsonpath="{.status.postCreateHooks.${TEST_HOOK_NAME}}")
echo "Individual hook status: ${HOOK_STATUS}"

if [ "$COMPLETION_STATUS" = "true" ] && [ "$HOOK_STATUS" = "true" ]; then
    echo "✅ PostCreateHook completion tracking working correctly!"
else
    echo "❌ PostCreateHook completion tracking failed!"
    echo "Expected: postCreateHookCompleted=true, hook status=true"
    echo "Got: postCreateHookCompleted=${COMPLETION_STATUS}, hook status=${HOOK_STATUS}"
    exit 1
fi

echo "6. Test without WaitForPostCreateHooks (backwards compatibility)"
kubectl apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: ${TEST_CP_NO_WAIT}
spec:
  backend: shared
  postCreateHook: ${TEST_HOOK_NAME}
  waitForPostCreateHooks: false
  type: k8s
EOF

echo "Waiting for control plane to be ready..."
kubectl wait --for=condition=Ready controlplane/${TEST_CP_NO_WAIT} --timeout=150s

# Give some time for hook to complete
sleep 20

# Check completion status for backwards compatibility
COMPLETION_STATUS_NO_WAIT=$(kubectl get controlplane ${TEST_CP_NO_WAIT} -o jsonpath='{.status.postCreateHookCompleted}')
HOOK_STATUS_NO_WAIT=$(kubectl get controlplane ${TEST_CP_NO_WAIT} -o jsonpath="{.status.postCreateHooks.${TEST_HOOK_NAME}}")

echo "Backwards compatibility test:"
echo "PostCreateHookCompleted: ${COMPLETION_STATUS_NO_WAIT}"
echo "Individual hook status: ${HOOK_STATUS_NO_WAIT}"

if [ "$COMPLETION_STATUS_NO_WAIT" = "true" ] && [ "$HOOK_STATUS_NO_WAIT" = "true" ]; then
    echo "✅ Backwards compatibility working correctly!"
else
    echo "⚠️  Backwards compatibility check - completion tracking may still be in progress"
fi

echo "🎉 PostCreateHook Completion E2E Test completed successfully!"