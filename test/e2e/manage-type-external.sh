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

:
: -------------------------------------------------------------------------
: Apply a post-create-hook for testing
:
kubectl --context kind-kubeflex apply -f ${SRC_DIR}/pod-pch.yaml

:
: -------------------------------------------------------------------------
: Create an external cluster to be adopted
:
kind create cluster --name ${EXT_CLUSTER_NAME}

:
: -------------------------------------------------------------------------
: Create a ControlPlane of type external
:

./bin/kflex adopt --adopted-context kind-${EXT_CLUSTER_NAME} --url-override https://${EXT_CLUSTER_NAME}-control-plane:6443 ${EXT_CLUSTER_NAME} --postcreate-hook list-controller --chatty-status=false

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
