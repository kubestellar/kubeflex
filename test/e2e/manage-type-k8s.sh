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
: Wait for the running components of ControlPlane cp1 to be ready, with
: default timeout which is 30 seconds
:
kubectl --context kind-kubeflex -n cp1-system wait --for=condition=Available deployment/kube-apiserver
kubectl --context kind-kubeflex -n cp1-system wait --for=condition=Available deployment/kube-controller-manager

:
: -------------------------------------------------------------------------
: Specify PostCreateHooks for cp1, then wait for the PostCreateHooks to
: take effect, with default timeout which is 30 seconds
:
kubectl --context kind-kubeflex patch cp/cp1 --type=merge --patch '{"spec":{"postCreateHooks":[{"hookName":"synthetic-crd"}]}}'
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
: Delete ControlPlane cp1
:
./bin/kflex delete cp1 --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type k8s
: