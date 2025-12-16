#!/usr/bin/env bash
# Copyright 2024 The KubeStellar Authors.
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

set -x # echo so that users can understand what is happening
set -e # exit on error

SRC_DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"
source "${SRC_DIR}/setup-shell.sh"

:
: -------------------------------------------------------------------------
: Test that controller manager image updates properly with make install-local-chart
: This test verifies the fix for issue #453 where pods weren't restarting
: with new images after running make install-local-chart
:

echo "=== Testing Controller Manager Image Update Fix ==="

# Verify kubeflex is installed and running
echo "1. Verifying kubeflex installation..."
if ! kubectl get namespace kubeflex-system >/dev/null 2>&1; then
    echo "ERROR: kubeflex-system namespace not found. Please install kubeflex first."
    exit 1
fi

# Get the current image before making changes
echo "2. Getting current controller manager image..."
CURRENT_IMAGE=$(kubectl get deployment kubeflex-controller-manager -n kubeflex-system -o jsonpath='{.spec.template.spec.containers[?(@.name=="manager")].image}')
if [ -z "$CURRENT_IMAGE" ]; then
    echo "ERROR: Could not get current image from deployment"
    exit 1
fi
echo "Current image: $CURRENT_IMAGE"

# Get current pod names to verify they change
echo "3. Getting current pod names..."
CURRENT_PODS=$(kubectl get pods -n kubeflex-system -l control-plane=controller-manager -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | sort | tr '\n' ' ')
echo "Current pods (sorted): $CURRENT_PODS"

# Set a unique image tag to force a change
echo "4. Setting unique image tag for testing..."
export IMAGE_TAG="e2e-test-$(date +%s)"
echo "Using image tag: $IMAGE_TAG"

# Run the make install-local-chart command
echo "5. Running make install-local-chart..."
make install-local-chart

# Wait for the deployment to update with proper timeout and status checking
echo "6. Waiting for deployment to update..."
if ! kubectl rollout status deployment/kubeflex-controller-manager -n kubeflex-system --timeout=180s; then
    echo "ERROR: Deployment rollout failed or timed out"
    echo "Deployment status:"
    kubectl describe deployment kubeflex-controller-manager -n kubeflex-system
    echo "Pod status:"
    kubectl get pods -n kubeflex-system -l control-plane=controller-manager
    exit 1
fi

# Wait for all the old pods to go away
echo "7. Wait for old Pods to go away"
while ! kubectl get pods -n kubeflex-system -l control-plane=controller-manager | wc -l | grep -qw 2; do
    echo Waiting for just one kubeflex-controller-manager Pod
    sleep 10
done

# Get the new image
echo "8. Getting new controller manager image..."
NEW_IMAGE=$(kubectl get deployment kubeflex-controller-manager -n kubeflex-system -o jsonpath='{.spec.template.spec.containers[?(@.name=="manager")].image}')
if [ -z "$NEW_IMAGE" ]; then
    echo "ERROR: Could not get new image from deployment"
    exit 1
fi
echo "New image: $NEW_IMAGE"

# Get new pod names
echo "9. Getting new pod names..."
NEW_PODS=$(kubectl get pods -n kubeflex-system -l control-plane=controller-manager -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | sort | tr '\n' ' ')
echo "New pods (sorted): $NEW_PODS"

# Wait for all pods to be ready
echo "10. Waiting for all pods to be ready..."
if ! kubectl wait --for=condition=Ready pods -l control-plane=controller-manager -n kubeflex-system --timeout=120s; then
    echo "ERROR: Not all pods are ready within timeout"
    echo "Pods:"
    kubectl get pods -n kubeflex-system -l control-plane=controller-manager --no-headers=true | while read ns name rest; do
	echo
	kubectl get pod -n $ns $name -o yaml
	echo
	kubectl events --namespace $ns --for pod/$name
    done
    exit 1
fi

# Verify the fix worked
echo "11. Verifying the fix..."

# Check if image changed
if [ "$CURRENT_IMAGE" = "$NEW_IMAGE" ]; then
    echo " FAILURE: Image did not change"
    echo "   Current: $CURRENT_IMAGE"
    echo "   New: $NEW_IMAGE"
    exit 1
fi

# Check if pods changed (new pods created) 
if [ "$CURRENT_PODS" = "$NEW_PODS" ]; then
    echo " FAILURE: Pods did not change (no new pods created)"
    echo " Current pods (sorted): $CURRENT_PODS"
    echo " New pods (sorted): $NEW_PODS"
    exit 1
fi

# Check if new image contains our test tag
if [[ "$NEW_IMAGE" != *"$IMAGE_TAG"* ]]; then
    echo " FAILURE: New image does not contain expected tag"
    echo "   Expected tag: $IMAGE_TAG"
    echo "   New image: $NEW_IMAGE"
    exit 1
fi

# Check if all pods are running
POD_STATUS=$(kubectl get pods -n kubeflex-system -l control-plane=controller-manager -o jsonpath='{.items[*].status.phase}')
if [[ "$POD_STATUS" != *"Running"* ]]; then
    echo " FAILURE: Pods are not in Running state"
    echo "   Pod status: $POD_STATUS"
    exit 1
fi

# Verify deployment is available
echo "12. Verifying deployment is available..."
if ! kubectl wait --for=condition=Available deployment/kubeflex-controller-manager -n kubeflex-system --timeout=60s; then
    echo " FAILURE: Deployment is not available"
    kubectl describe deployment kubeflex-controller-manager -n kubeflex-system
    exit 1
fi

echo "   SUCCESS: Controller manager image update test passed!"
echo "   Image changed from '$CURRENT_IMAGE' to '$NEW_IMAGE'"
echo "   New pods created: $NEW_PODS"
echo "   All pods are running"
echo "   Deployment is available"

:
: -------------------------------------------------------------------------
: SUCCESS: Controller manager image updates properly with make install-local-chart
:
