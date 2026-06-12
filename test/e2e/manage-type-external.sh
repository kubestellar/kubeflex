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

# Currently this only works whe testing with a host cluster made my `kind`.
platform="kind"

SRC_DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"
source "${SRC_DIR}/setup-shell.sh"

set -x # echo so that users can understand what is happening
set -e # exit on error

host_context="${platform}-kubeflex"
EXT_CLUSTER_NAME=ext

:
: -------------------------------------------------------------------------
: Define a post-create-hook for testing
:
kubectl --context "$host_context" apply -f ${SRC_DIR}/list-controller-pch.yaml

:
: -------------------------------------------------------------------------
: Create an external cluster to be adopted
:
if [ "$platform" == kind ]; then
    kind create cluster --name ${EXT_CLUSTER_NAME}
    override_endpoint="${EXT_CLUSTER_NAME}-control-plane:6443"
else
    k3d cluster create --image rancher/k3s:v1.32.13-k3s1 -p "10443:443@loadbalancer" --k3s-arg "--disable=traefik@server:*" ${EXT_CLUSTER_NAME}
    override_endpoint="k3d-${EXT_CLUSTER_NAME}-server-0:6443"
fi

:
: -------------------------------------------------------------------------
: Create a ControlPlane of type external
:

./bin/kflex adopt --adopted-context "${platform}-${EXT_CLUSTER_NAME}" --url-override "https://${override_endpoint}" ${EXT_CLUSTER_NAME} --postcreate-hook list-controller --chatty-status=false

:
: -------------------------------------------------------------------------
: Wait for adopted cluster secret to be created
:
wait-for-secret "$host_context" ${EXT_CLUSTER_NAME}-system admin-kubeconfig

:
: -------------------------------------------------------------------------
: Wait for test controller pod installed by pch to be running
:

kubectl --context "$host_context" -n ${EXT_CLUSTER_NAME}-system wait --for=condition=Ready pod list-controller

:
: -------------------------------------------------------------------------
: Delete ControlPlane 
:
./bin/kflex delete ${EXT_CLUSTER_NAME} --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type external
:
