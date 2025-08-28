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

CP_NAME="k3s-cp"
CP_TYPE="k3s"
:
: -------------------------------------------------------------------------
: Create a ControlPlane of type k3s
:
./bin/kflex create $CP_NAME --type $CP_TYPE --chatty-status=false

:
: -------------------------------------------------------------------------
: Verify that the kubeconfig has the correct extensions structure
: with controlplane-name set in the context
:
echo "Verifying kubeconfig extensions for control plane $CP_NAME..."

# Extract the controlplane-name from kubeconfig extensions using yq
actual_cp_name=$(kubectl config view -o=yaml | yq -r '.contexts[] | select(.name == "'$CP_NAME'") | .context.extensions[] | select(.name == "kubeflex") | .extension.data["controlplane-name"] // ""')

if [ -z "$actual_cp_name" ]; then
    echo "ERROR: Context $CP_NAME not found or does not have kubeflex extension with controlplane-name"
    echo "Available contexts:"
    kubectl config view -o=yaml | yq -r '.contexts[].name'
    exit 1
fi

if [ "$actual_cp_name" != "$CP_NAME" ]; then
    echo "ERROR: Expected controlplane-name '$CP_NAME', but found '$actual_cp_name'"
    exit 1
fi

echo "SUCCESS: Kubeconfig extensions verified for control plane $CP_NAME"

:
: -------------------------------------------------------------------------
: Wait for the running component of ControlPlane $CP_NAME to be ready/completed
:
kubectl --context kind-kubeflex -n $CP_NAME-system wait --for=jsonpath='{.status.availableReplicas}'=1 statefulset/k3s-server --timeout=120s
kubectl --context kind-kubeflex -n $CP_NAME-system wait --for=condition=Complete job/k3s-bootstrap-kubeconfig --timeout=120s
kubectl --context kind-kubeflex -n $CP_NAME-system wait --for=jsonpath='{.status.conditions[?(@.type == "Ready")]}' cps/$CP_NAME --timeout=120s

: -------------------------------------------------------------------------
: Create a Deployment in ControlPlane $CP_NAME, then wait for the Deployment
: to become available
:
kubectl --context $CP_NAME create deployment my-nginx --image public.ecr.aws/nginx/nginx:1.26.3

kubectl --context $CP_NAME wait --for=condition=Available deploy/my-nginx --timeout=120s

:
: -------------------------------------------------------------------------
: Delete ControlPlane $CP_NAME
:
./bin/kflex delete $CP_NAME --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type k3s
:
