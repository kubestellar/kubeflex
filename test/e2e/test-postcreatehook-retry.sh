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

CP_TYPE=${1:-k8s}
echo "Testing PostCreateHook retry logic with ${CP_TYPE} control plane..."

SRC_DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"
source "${SRC_DIR}/setup-shell.sh"

set -x # echo commands for better debugging
set -e # exit on error

:
: -------------------------------------------------------------------------
: Clean up any existing test resources
:
echo "Cleaning up any existing resources..."
kubectl --context kind-kubeflex delete controlplane cp-missing-hook-${CP_TYPE} --ignore-not-found=true
kubectl --context kind-kubeflex delete postcreatehook missing-hook-${CP_TYPE} --ignore-not-found=true

:
: -------------------------------------------------------------------------
: Test case: ControlPlane with missing PostCreateHook should not fail
: but should retry until the hook is available
:
echo "Creating ControlPlane referencing a missing PostCreateHook..."
kubectl --context kind-kubeflex apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: cp-missing-hook-${CP_TYPE}
spec:
  backend: shared
  postCreateHook: missing-hook-${CP_TYPE}
  waitForPostCreateHooks: true
  type: ${CP_TYPE}
EOF

:
: -------------------------------------------------------------------------
: Verify ControlPlane is not marked as failed while waiting for hook
:
echo "Waiting 10s to check that ControlPlane is not marked as failed..."
sleep 10

echo "ControlPlane status after 10s (should NOT be failed):"
kubectl --context kind-kubeflex get controlplane cp-missing-hook-${CP_TYPE} -o jsonpath='{.status.conditions}' | jq '.'

:
: -------------------------------------------------------------------------
: Create the missing PostCreateHook
:
echo "Creating the missing PostCreateHook..."
kubectl --context kind-kubeflex apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: missing-hook-${CP_TYPE}
spec:
  templates:
  - apiVersion: batch/v1
    kind: Job
    metadata:
      name: job-missing-hook-{{.ControlPlaneName}}
    spec:
      template:
        spec:
          containers:
          - name: demo
            image: public.ecr.aws/docker/library/busybox:1.36
            command: ["echo", "Hello from missing hook for ${CP_TYPE}"]
          restartPolicy: Never
      backoffLimit: 1
EOF

:
: -------------------------------------------------------------------------
: Verify ControlPlane becomes Ready after hook is created
:
echo "Waiting for ControlPlane to become Ready (180s timeout)..."
kubectl --context kind-kubeflex wait --for=condition=Ready controlplane/cp-missing-hook-${CP_TYPE} --timeout=180s

echo "FINAL STATUS:"
kubectl --context kind-kubeflex get controlplane cp-missing-hook-${CP_TYPE} -o jsonpath='{.status}' | jq '.'

:
: -------------------------------------------------------------------------
: Clean up test resources
:
echo "Cleaning up test resources..."
kubectl --context kind-kubeflex delete controlplane cp-missing-hook-${CP_TYPE} --ignore-not-found=true
kubectl --context kind-kubeflex delete postcreatehook missing-hook-${CP_TYPE} --ignore-not-found=true

:
: -------------------------------------------------------------------------
: SUCCESS: Verified retry logic for missing PostCreateHook
:
echo "SUCCESS: ${CP_TYPE} PostCreateHook retry test completed"
