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
./bin/kflex create mysupercp --type k8s --chatty-status=false

:
: -------------------------------------------------------------------------
: Verify that the kubeconfig has the correct extensions structure
: with controlplane-name set in the context
:
echo "Verifying kubeconfig extensions for control plane mysupercp..."

# Check that the context exists
if ! kubectl config view --raw | grep -q "name: mysupercp"; then
    echo "ERROR: Context mysupercp not found in kubeconfig"
    exit 1
fi

# Check that the context has the kubeflex extension with controlplane-name
if ! kubectl config view --raw | grep -A 50 "name: mysupercp" | grep -A 20 "extensions:" | grep -q "controlplane-name: mysupercp"; then
    echo "ERROR: Context mysupercp does not have controlplane-name extension set to mysupercp"
    echo "Showing the mysupercp context section:"
    kubectl config view --raw | grep -A 30 "name: mysupercp"
    exit 1
fi

echo "SUCCESS: Kubeconfig extensions verified for control plane mysupercp"

:
: -------------------------------------------------------------------------
: Wait for the running components of ControlPlane mysupercp to be ready, with
: default timeout which is 30 seconds
:
kubectl --context kind-kubeflex -n mysupercp-system wait --for=condition=Available deployment/kube-apiserver
kubectl --context kind-kubeflex -n mysupercp-system wait --for=condition=Available deployment/kube-controller-manager

:
: -------------------------------------------------------------------------
: Specify a PostCreateHook for mysupercp, then wait for the PostCreateHook to
: take effect, with default timeout which is 30 seconds
:
kubectl --context kind-kubeflex patch cp/mysupercp --type=merge --patch '{"spec":{"postCreateHook":"synthetic-crd"}}'
wait-for-cmd 'kubectl --context kind-kubeflex get crd cr1s.synthetic-crd.com'
kubectl --context kind-kubeflex wait --for=condition=Established crd cr1s.synthetic-crd.com

:
: -------------------------------------------------------------------------
: Create a namespace in ControlPlane mysupercp, then wait for the namespace to
: become active
:
kubectl --context mysupercp create ns e2e
kubectl --context mysupercp wait --for=jsonpath='{.status.phase}'=Active ns/e2e --timeout=120s

:
: -------------------------------------------------------------------------
: Delete ControlPlane mysupercp
:
./bin/kflex delete mysupercp --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type k8s
:
