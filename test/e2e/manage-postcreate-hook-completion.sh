#!/usr/bin/env bash

# Copyright 2025 The KubeStellar Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Navigate to test directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

echo "=== Testing PostCreateHook Completion Feature ==="

# Use hosting cluster context
kubectl config use-context kind-kubeflex

# Apply the PostCreateHook
echo "Creating PostCreateHook..."
kubectl apply -f postcreate-hook-completion-test.yaml

# Create control plane with PostCreateHook and WaitForPostCreateHooks enabled
echo "Creating control plane with PostCreateHook completion testing..."
../../bin/kflex delete completion-test-cp --chatty-status=false 2>/dev/null || true

# Create ControlPlane with waitForPostCreateHooks enabled
cat <<EOF | kubectl apply -f -
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: completion-test-cp
spec:
  type: k8s
  backend: shared
  postCreateHook: completion-test-hook
  waitForPostCreateHooks: true
EOF

echo "Waiting for control plane to be created..."
sleep 5

# Monitor the PostCreateHook completion status
echo "Monitoring PostCreateHook completion status..."
timeout=180
start_time=$(date +%s)

while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    if [ $elapsed -gt $timeout ]; then
        echo "❌ Timeout waiting for PostCreateHook completion"
        exit 1
    fi
    
    # Check only 2 things: hook applied and CP ready
    hook_completed=$(kubectl get cp completion-test-cp -o jsonpath='{.status.postCreateHooks.completion-test-hook}' 2>/dev/null || echo 'false')
    cp_ready=$(kubectl get cp completion-test-cp -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo 'False')
    
    echo "[$elapsed s] Hook applied: $hook_completed, CP ready: $cp_ready"
    
    # Check if hook is applied
    if [ "$hook_completed" = "true" ]; then
        echo "✅ PostCreateHook successfully applied"
        
        # Control plane should be ready after hook is applied
        if [ "$cp_ready" = "True" ]; then
            echo "✅ Control plane is ready after PostCreateHook completion"
            break
        fi
    fi
    
    sleep 5
done

# Validate the created resources
echo "Validating created resources..."
kubectl get all -n completion-test-cp-system

# Check job completion
job_status=$(kubectl get job completion-test-job -n completion-test-cp-system -o jsonpath='{.status.succeeded}' 2>/dev/null || echo '0')
if [ "$job_status" -gt 0 ]; then
    echo "✅ Job completed successfully"
else
    echo "❌ Job did not complete"
    exit 1
fi

# Check deployment readiness
deployment_ready=$(kubectl get deployment completion-test-deployment -n completion-test-cp-system -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo '0')
if [ "$deployment_ready" -gt 0 ]; then
    echo "✅ Deployment is ready"
else
    echo "❌ Deployment is not ready"
    exit 1
fi

# Test without WaitForPostCreateHooks (backwards compatibility)
echo "Testing backwards compatibility (waitForPostCreateHooks=false)..."
../../bin/kflex delete completion-test-cp-compat --chatty-status=false 2>/dev/null || true

cat <<EOF | kubectl apply -f -
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: completion-test-cp-compat
spec:
  type: k8s
  backend: shared
  postCreateHook: completion-test-hook
  waitForPostCreateHooks: false
EOF

echo "Waiting for control plane (compat mode) to be ready..."
kubectl wait --for=condition=Ready cp/completion-test-cp-compat --timeout=120s

hook_completed_compat=$(kubectl get cp completion-test-cp-compat -o jsonpath='{.status.postCreateHooks.completion-test-hook}' 2>/dev/null || echo 'false')
if [ "$hook_completed_compat" = "true" ]; then
    echo "✅ PostCreateHook applied in compatibility mode"
else
    echo "❌ PostCreateHook not applied in compatibility mode"
    exit 1
fi

# Cleanup
echo "Cleaning up..."
../../bin/kflex delete completion-test-cp --chatty-status=false 2>/dev/null || true
../../bin/kflex delete completion-test-cp-compat --chatty-status=false 2>/dev/null || true
kubectl delete postcreatehook completion-test-hook

echo "🎉 PostCreateHook completion feature test passed!" 