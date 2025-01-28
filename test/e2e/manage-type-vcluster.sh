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

:
: -------------------------------------------------------------------------
: Create a ControlPlane of type vcluster
:
./bin/kflex create cp2 --type vcluster --chatty-status=false

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
kubectl --context cp2 create deployment my-nginx --image nginx 
kubectl --context cp2 wait --for=condition=Available deploy/my-nginx --timeout=120s

:
: -------------------------------------------------------------------------
: Delete ControlPlane cp2
:
./bin/kflex delete cp2 --chatty-status=false

:
: -------------------------------------------------------------------------
: SUCCESS: manage a ControlPlane of type vcluster
:
