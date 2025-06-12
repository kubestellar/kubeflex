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

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

:
: -------------------------------------------------------------------------
: Create the hosting kind cluster with ingress controller
:

kind create cluster --name kubeflex --config ${SCRIPT_DIR}/kind-config.yaml

# Install ingress-nginx using kubectl apply
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/refs/tags/controller-v1.12.1/deploy/static/provider/kind/deploy.yaml
kubectl -n ingress-nginx patch deployment/ingress-nginx-controller --patch-file=${SCRIPT_DIR}/nginx-patch.yaml

# Wait for ingress controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

:
: -------------------------------------------------------------------------
: Compile binaries
:
make build

:
: -------------------------------------------------------------------------
: Build the OCI image for the kubeflex controller manager and load it in the local docker registry
:
:
make ko-local-build

# -------------------------------------------------------------------------
# Ensure kubeflex-system namespace exists
# -------------------------------------------------------------------------
kubectl create namespace kubeflex-system --dry-run=client -o yaml | kubectl apply -f -

make install-local-chart

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

:
: -------------------------------------------------------------------------
: Wait for kubeflex-controller-manager ready
:

kubectl wait --for=condition=available --timeout=300s -n kubeflex-system deployment/kubeflex-controller-manager
