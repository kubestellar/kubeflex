#!/usr/bin/env bash

# Copyright 2023 The KubeStellar Authors.
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

# This script generates typed clients, informers, and listers for the
# kubeflex CRDs using the Kubernetes code-generator.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${SCRIPT_ROOT}"

# Module and package configuration
MODULE="github.com/kubestellar/kubeflex"
API_PKG="api"
OUTPUT_PKG="pkg/generated"
GROUP_VERSION="tenancy:v1alpha1"

# Determine code-generator version from go.mod
CODEGEN_VERSION=$(go list -m -f '{{.Version}}' k8s.io/code-generator 2>/dev/null || echo "v0.32.10")

# Create a temporary directory for code-generator
CODEGEN_PKG="${GOPATH:-$HOME/go}/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION}"

# If code-generator is not available, install it
if [[ ! -d "${CODEGEN_PKG}" ]]; then
    echo "Installing k8s.io/code-generator@${CODEGEN_VERSION}..."
    go install "k8s.io/code-generator/cmd/client-gen@${CODEGEN_VERSION}"
    go install "k8s.io/code-generator/cmd/lister-gen@${CODEGEN_VERSION}"
    go install "k8s.io/code-generator/cmd/informer-gen@${CODEGEN_VERSION}"
fi

# Header file for generated code
BOILERPLATE="${SCRIPT_ROOT}/hack/boilerplate.go.txt"

echo "Generating clientset..."
go run k8s.io/code-generator/cmd/client-gen@${CODEGEN_VERSION} \
    --clientset-name "versioned" \
    --input-base "${MODULE}/${API_PKG}" \
    --input "v1alpha1" \
    --output-dir "${SCRIPT_ROOT}/${OUTPUT_PKG}/clientset" \
    --output-pkg "${MODULE}/${OUTPUT_PKG}/clientset" \
    --go-header-file "${BOILERPLATE}"

echo "Generating listers..."
go run k8s.io/code-generator/cmd/lister-gen@${CODEGEN_VERSION} \
    --output-dir "${SCRIPT_ROOT}/${OUTPUT_PKG}/listers" \
    --output-pkg "${MODULE}/${OUTPUT_PKG}/listers" \
    --go-header-file "${BOILERPLATE}" \
    "${MODULE}/${API_PKG}/v1alpha1"

echo "Generating informers..."
go run k8s.io/code-generator/cmd/informer-gen@${CODEGEN_VERSION} \
    --versioned-clientset-package "${MODULE}/${OUTPUT_PKG}/clientset/versioned" \
    --listers-package "${MODULE}/${OUTPUT_PKG}/listers" \
    --output-dir "${SCRIPT_ROOT}/${OUTPUT_PKG}/informers" \
    --output-pkg "${MODULE}/${OUTPUT_PKG}/informers" \
    --go-header-file "${BOILERPLATE}" \
    "${MODULE}/${API_PKG}/v1alpha1"

echo "Code generation complete."
