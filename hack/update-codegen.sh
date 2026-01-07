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

# Fail if working tree is not clean
if [[ -n "$(git status --porcelain)" ]]; then
    echo "Error: working tree is not clean"
    git status
    exit 1
fi

# Module and package configuration
MODULE="github.com/kubestellar/kubeflex"
API_PKG="api"
OUTPUT_PKG="pkg/generated"

# Determine code-generator version from go.mod
CODEGEN_VERSION=$(go list -m -f '{{.Version}}' k8s.io/code-generator)

# Tools directory (repo-local, not GOPATH)
TOOLS_BIN_DIR="${SCRIPT_ROOT}/hack/tools/bin"
mkdir -p "${TOOLS_BIN_DIR}"
TOOLS_BIN_DIR="$(cd "${TOOLS_BIN_DIR}" && pwd)"

# Install code generators to repo-local directory
echo "Installing code-generator tools to ${TOOLS_BIN_DIR}..."
GOBIN="${TOOLS_BIN_DIR}" go install "k8s.io/code-generator/cmd/client-gen@${CODEGEN_VERSION}"
GOBIN="${TOOLS_BIN_DIR}" go install "k8s.io/code-generator/cmd/lister-gen@${CODEGEN_VERSION}"
GOBIN="${TOOLS_BIN_DIR}" go install "k8s.io/code-generator/cmd/informer-gen@${CODEGEN_VERSION}"

# Header file for generated code
BOILERPLATE="${SCRIPT_ROOT}/hack/boilerplate.go.txt"

# Delete entire generated code tree for reproducibility
echo "Clearing ${OUTPUT_PKG}..."
rm -rf "${SCRIPT_ROOT}/${OUTPUT_PKG}"

echo "Generating clientset..."
"${TOOLS_BIN_DIR}/client-gen" \
    --clientset-name "versioned" \
    --input-base "${MODULE}/${API_PKG}" \
    --input "v1alpha1" \
    --output-dir "${SCRIPT_ROOT}/${OUTPUT_PKG}/clientset" \
    --output-pkg "${MODULE}/${OUTPUT_PKG}/clientset" \
    --go-header-file "${BOILERPLATE}"

echo "Generating listers..."
"${TOOLS_BIN_DIR}/lister-gen" \
    --output-dir "${SCRIPT_ROOT}/${OUTPUT_PKG}/listers" \
    --output-pkg "${MODULE}/${OUTPUT_PKG}/listers" \
    --go-header-file "${BOILERPLATE}" \
    "${MODULE}/${API_PKG}/v1alpha1"

echo "Generating informers..."
"${TOOLS_BIN_DIR}/informer-gen" \
    --versioned-clientset-package "${MODULE}/${OUTPUT_PKG}/clientset/versioned" \
    --listers-package "${MODULE}/${OUTPUT_PKG}/listers" \
    --output-dir "${SCRIPT_ROOT}/${OUTPUT_PKG}/informers" \
    --output-pkg "${MODULE}/${OUTPUT_PKG}/informers" \
    --go-header-file "${BOILERPLATE}" \
    "${MODULE}/${API_PKG}/v1alpha1"

echo "Code generation complete."
