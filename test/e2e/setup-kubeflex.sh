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
release=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --release)
      if [[ $# -lt 2 ]]; then
        echo "Error: --release requires a value (e.g. v0.9.2 or latest)"
        exit 1
      fi
      release="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)

:
: -------------------------------------------------------------------------
: Create the hosting kind cluster with ingress controller
:

kind create cluster --name kubeflex --config ${SCRIPT_DIR}/kind-config.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/refs/tags/controller-v1.12.1/deploy/static/provider/kind/deploy.yaml
kubectl -n ingress-nginx patch deployment/ingress-nginx-controller --patch-file=${SCRIPT_DIR}/nginx-patch.yaml

:
: -------------------------------------------------------------------------
if [[ -z "${release}" ]]; then
    echo "Installing kubeflex from local source"
    : Compile binaries
    :
    make build

    :
    : -------------------------------------------------------------------------
    : Build the OCI image for the kubeflex controller manager and load it in the local docker registry
    :
    :
    make ko-local-build

    :
    : -------------------------------------------------------------------------
    : Load the local image in kind, re-generate manifests and helm chart, and install the helm chart:
    :
    :
    make install-local-chart
    :
    :
else
    if [[ "${release}" == "latest" ]]; then
        echo "Resolving latest kubeflex release from GitHub"
        release="$(curl -fsSL https://api.github.com/repos/kubestellar/kubeflex/releases/latest \
          | jq -r '.tag_name')"

        if [[ -z "${release}" ]]; then
            echo "Failed to resolve latest kubeflex release"
            exit 1
        fi

        echo "Resolved latest release to ${release}"
    fi
    echo "Installing kubeflex release ${release}"

    bash <(curl -s https://raw.githubusercontent.com/kubestellar/kubeflex/refs/tags/${release}/scripts/install-kubeflex.sh) --version $release --ensure-folder bin --strip-bin -X

    if [[ "${release}" < v0.9.2 ]]; then
      kubectl create namespace kubeflex-system --dry-run=client -o yaml | kubectl apply -f -
      helm install kubeflex-operator \
        oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator \
        --namespace kubeflex-system \
        --version "${release}"
    else
      helm install kubeflex-operator \
        oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator \
        --version "${release}"
    fi

fi
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
