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

set -x # echo so that users can understand what is happening
set -e # exit on error

EXT_CLUSTER_NAME=ext
cluster_type="kind"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cluster-type)
      if (( $# < 2 )); then
        echo "Error: --cluster-type requires a value (kind or k3d)" >&2
        exit 1
      fi
      cluster_type="$2"
      if [[ "${cluster_type}" != "kind" && "${cluster_type}" != "k3d" ]]; then
        echo "Error: --cluster-type must be 'kind' or 'k3d', got '${cluster_type}'" >&2
        exit 1
      fi
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done 

:
: -------------------------------------------------------------------------
: Define a post-create-hook for testing
:
kubectl --context kind-kubeflex apply -f ${SRC_DIR}/list-controller-pch.yaml

:
: -------------------------------------------------------------------------
: Create an external cluster to be adopted and adopt it
:
if [[ "${cluster_type}" == "k3d" ]]; then
    k3d cluster create ${EXT_CLUSTER_NAME} --network k3d-kubeflex
    kubectl config rename-context k3d-${EXT_CLUSTER_NAME} kind-${EXT_CLUSTER_NAME} || true
    ./bin/kflex adopt --adopted-context kind-${EXT_CLUSTER_NAME} --url-override https://k3d-${EXT_CLUSTER_NAME}-server-0:6443 ${EXT_CLUSTER_NAME} --postcreate-hook list-controller --chatty-status=false
else
    kind create cluster --name ${EXT_CLUSTER_NAME}
    ./bin/kflex adopt --adopted-context kind-${EXT_CLUSTER_NAME} --url-override https://${EXT_CLUSTER_NAME}-control-plane:6443 ${EXT_CLUSTER_NAME} --postcreate-hook list-controller --chatty-status=false
fi

:
: -------------------------------------------------------------------------
: Wait for adopted cluster secret to be created
:
wait-for-secret kind-kubeflex ${EXT_CLUSTER_NAME}-system admin-kubeconfig

:
: -------------------------------------------------------------------------
: Wait for test controller pod installed by pch to be running
:

kubectl --context kind-kubeflex -n ${EXT_CLUSTER_NAME}-system wait --for=condition=Ready pod list-controller

:
: -------------------------------------------------------------------------
: Delete ControlPlane 
:
./bin/kflex delete ${EXT_CLUSTER_NAME} --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type external
:
