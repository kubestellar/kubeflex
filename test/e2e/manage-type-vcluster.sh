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
: Create a ControlPlane of type vcluster
:
./bin/kflex create cp2 --type vcluster --chatty-status=false

:
: -------------------------------------------------------------------------
: Verify that the kubeconfig has the correct extensions structure
: with controlplane-name set in the context
:
echo "Verifying kubeconfig extensions for control plane cp2..."

context_name="cp2"
cp_name="cp2"

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

echo "SUCCESS: Kubeconfig extensions verified for control plane cp2"

:
: -------------------------------------------------------------------------
: Wait for the running component of ControlPlane cp2 to be ready/completed
:
kubectl --context kind-kubeflex -n cp2-system wait --for=jsonpath='{.status.availableReplicas}'=1 statefulset/vcluster --timeout=120s
kubectl --context kind-kubeflex -n cp2-system wait --for=condition=Complete job/update-cluster-info --timeout=120s

:
: -------------------------------------------------------------------------
: Create a Deployment in ControlPlane cp2, then wait for the Deployment
: to become available
:
kubectl --context cp2 create deployment my-nginx --image public.ecr.aws/nginx/nginx:1.26.3

kubectl --context cp2 wait --for=condition=Available deploy/my-nginx --timeout=120s

:
: -------------------------------------------------------------------------
: Test PostCreateHook in-cluster kubeconfig access
:
${SRC_DIR}/test-kubeconfig-access.sh -t vcluster

:
: -------------------------------------------------------------------------
: Delete ControlPlane cp2
:
./bin/kflex delete cp2 --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type vcluster
:
