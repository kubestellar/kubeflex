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
: Create a ControlPlane of type k8s
:
./bin/kflex create cp1 --type k8s --chatty-status=false

:
: -------------------------------------------------------------------------
: Verify that the kubeconfig has the correct extensions structure
: with controlplane-name set in the context
:
echo "Verifying kubeconfig extensions for control plane cp1..."

context_name="cp1"
cp_name="cp1"

# Extract the controlplane-name from kubeconfig extensions using yq
actual_cp_name=$(kubectl config view -o=yaml | yq -r '.contexts[] | select(.name == "'$context_name'") | .context.extensions[] | select(.name == "kubeflex") | .extension.data["controlplane-name"] // ""')

if [ -z "$actual_cp_name" ]; then
    echo "ERROR: Context $context_name not found or does not have kubeflex extension with controlplane-name"
    echo "Available contexts:"
    kubectl config view -o=yaml | yq -r '.contexts[].name'
    exit 1
fi

if [ "$actual_cp_name" != "$cp_name" ]; then
    echo "ERROR: Expected controlplane-name '$cp_name', but found '$actual_cp_name'"
    exit 1
fi

echo "SUCCESS: Kubeconfig extensions verified for control plane cp1"

:
: -------------------------------------------------------------------------
: Wait for the running components of ControlPlane cp1 to be ready, with
: default timeout which is 30 seconds
:
kubectl --context kind-kubeflex -n cp1-system wait --for=condition=Available deployment/kube-apiserver
kubectl --context kind-kubeflex -n cp1-system wait --for=condition=Available deployment/kube-controller-manager

:
: -------------------------------------------------------------------------
: Specify a PostCreateHook for cp1, then wait for the PostCreateHook to
: take effect, with default timeout which is 30 seconds
:
kubectl --context kind-kubeflex patch cp/cp1 --type=merge --patch '{"spec":{"postCreateHook":"synthetic-crd"}}'
wait-for-cmd 'kubectl --context kind-kubeflex get crd cr1s.synthetic-crd.com'
kubectl --context kind-kubeflex wait --for=condition=Established crd cr1s.synthetic-crd.com

:
: -------------------------------------------------------------------------
: Create a namespace in ControlPlane cp1, then wait for the namespace to
: become active
:
kubectl --context cp1 create ns e2e
kubectl --context cp1 wait --for=jsonpath='{.status.phase}'=Active ns/e2e --timeout=120s

:
: -------------------------------------------------------------------------
: Test PostCreateHook kubeconfig access
:
${SRC_DIR}/test-kubeconfig-access.sh k8s

:
: -------------------------------------------------------------------------
: Delete ControlPlane cp1
:
./bin/kflex delete cp1 --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type k8s
:
