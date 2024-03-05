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
: Compile binaries
:
make build

:
: -------------------------------------------------------------------------
: Create the hosting kind cluster with ingress controller and install
: the kubeflex operator
:
./bin/kflex init --create-kind --chatty-status=false

:
: -------------------------------------------------------------------------
: Create a PostCreateHook
:
kubectl --context kind-kubeflex apply -f - <<EOF
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: PostCreateHook
metadata:
  name: synthetic-crd
spec:
  templates:
  - apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    metadata:
      name: cr1s.synthetic-crd.com
    spec:
      group: synthetic-crd.com
      names:
        kind: CR1
        listKind: CR1List
        plural: cr1s
        singular: cr1
      scope: Namespaced
      versions:
      - name: v1alpha1
        served: true
        storage: true
        schema:
          openAPIV3Schema:
            type: object
            properties:
              spec:
                type: object
                properties:
                  tier:
                    type: string
                    enum:
                    - Dedicated
                    - Shared
                    default: Shared
              status:
                type: object
                properties:
                  phase:
                    type: string
            required:
            - spec
        subresources:
          status: {}
EOF
