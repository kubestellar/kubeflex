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

SRC_DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"
source "${SRC_DIR}/setup-shell.sh"

# Accept control plane type as parameter
CP_TYPE=${1:-k8s}
CP_NAME="kubeconfig-test-${CP_TYPE}"

set -e # exit on error
set -x # for debugging

echo "Testing kubeconfig access via PostCreateHook for ${CP_TYPE}..."

# Get the appropriate secret name for the CP type
case $CP_TYPE in
  k8s)
    SECRET_NAME="admin-kubeconfig"
    SECRET_KEY="kubeconfig-incluster"
    ;;
  vcluster)
    SECRET_NAME="vc-vcluster"
    SECRET_KEY="config-incluster"
    ;;
  *)
    echo "Unsupported CP type: $CP_TYPE (only k8s and vcluster supported)"
    exit 1
    ;;
esac

echo "Using secret: $SECRET_NAME with key: $SECRET_KEY for $CP_TYPE"

:
: -------------------------------------------------------------------------
: Clean up any existing resources
:
kubectl --context kind-kubeflex delete controlplane ${CP_NAME} --ignore-not-found=true
kubectl --context kind-kubeflex delete postcreatehook kubeconfig-test-${CP_TYPE} --ignore-not-found=true

:
: -------------------------------------------------------------------------
: Create PostCreateHook that tests kubeconfig access
:
kubectl --context kind-kubeflex apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: kubeconfig-test-${CP_TYPE}
spec:
  templates:
  - apiVersion: batch/v1
    kind: Job
    metadata:
      name: kubeconfig-test-{{.ControlPlaneName}}
    spec:
      template:
        spec:
          containers:
          - name: kubeconfig-tester
            image: quay.io/kubestellar/kubectl:1.30.12
            command: ["/bin/sh", "-c"]
            args:
            - |
              if kubectl --kubeconfig=/root/.kube/${SECRET_KEY} get namespace kube-system > /dev/null 2>&1; then
                echo "SUCCESS: Can access ControlPlane API server"
                exit 0
              else
                echo "FAILED: Cannot access ControlPlane API server"
                exit 1
              fi
            volumeMounts:
            - name: kubeconfig-volume
              mountPath: "/root/.kube"
          restartPolicy: Never
          volumes:
          - name: kubeconfig-volume
            secret:
              secretName: ${SECRET_NAME}
      backoffLimit: 3
EOF

:
: -------------------------------------------------------------------------
: Create ControlPlane with kubeconfig test PostCreateHook
:
echo "Creating ${CP_TYPE} ControlPlane..."
kubectl --context kind-kubeflex apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: ${CP_NAME}
spec:
  backend: shared
  postCreateHook: kubeconfig-test-${CP_TYPE}
  waitForPostCreateHooks: true
  type: ${CP_TYPE}
EOF

:
: -------------------------------------------------------------------------
: Wait for ControlPlane to be ready
:
echo "Waiting for ${CP_TYPE} ControlPlane to be ready..."
kubectl --context kind-kubeflex wait --for=condition=Ready controlplane/${CP_NAME} --timeout=150s

:
: -------------------------------------------------------------------------
: Verify PostCreateHook Job completed successfully
:
kubectl --context kind-kubeflex wait --for=condition=Complete job/kubeconfig-test-${CP_NAME} -n ${CP_NAME}-system --timeout=60s

:
: -------------------------------------------------------------------------
: Clean up test resources
:
kubectl --context kind-kubeflex delete controlplane ${CP_NAME} --ignore-not-found=true
kubectl --context kind-kubeflex delete postcreatehook kubeconfig-test-${CP_TYPE} --ignore-not-found=true

echo "SUCCESS: ${CP_TYPE} PostCreateHook kubeconfig access test completed"
