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
./bin/kflex create myhellocp --type vcluster --chatty-status=false

:
: -------------------------------------------------------------------------
: Verify that the kubeconfig has the correct extensions structure
: with controlplane-name set in the context
:
echo "Verifying kubeconfig extensions for control plane myhellocp..."

# Check that the context exists
if ! kubectl config view --raw | grep -q "name: myhellocp"; then
    echo "ERROR: Context myhellocp not found in kubeconfig"
    exit 1
fi

# Check that the context has the kubeflex extension with controlplane-name
if ! kubectl config view --raw | grep -A 50 "name: myhellocp" | grep -A 20 "extensions:" | grep -q "controlplane-name: myhellocp"; then
    echo "ERROR: Context myhellocp does not have controlplane-name extension set to myhellocp"
    echo "Showing the myhellocp context section:"
    kubectl config view --raw | grep -A 30 "name: myhellocp"
    exit 1
fi

echo "SUCCESS: Kubeconfig extensions verified for control plane myhellocp"

:
: -------------------------------------------------------------------------
: Wait for the running component of ControlPlane myhellocp to be ready/completed
:
kubectl --context kind-kubeflex -n myhellocp-system wait --for=jsonpath='{.status.availableReplicas}'=1 statefulset/vcluster --timeout=120s
kubectl --context kind-kubeflex -n myhellocp-system wait --for=condition=Complete job/update-cluster-info --timeout=120s

:
: -------------------------------------------------------------------------
: Create a Deployment in ControlPlane myhellocp, then wait for the Deployment
: to become available
:
kubectl --context myhellocp create deployment my-nginx --image public.ecr.aws/nginx/nginx:1.26.3

kubectl --context myhellocp wait --for=condition=Available deploy/my-nginx --timeout=120s

:
: -------------------------------------------------------------------------
: Delete ControlPlane myhellocp
:
./bin/kflex delete myhellocp --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type vcluster
:
