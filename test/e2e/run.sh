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

# Change to repository root directory to ensure scripts work from any location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Verify we found the repository root
if [[ ! -f "${REPO_ROOT}/go.mod" ]]; then
    echo "Error: Could not find repository root (go.mod not found in ${REPO_ROOT})"
    exit 1
fi

cd "${REPO_ROOT}"
echo "Running E2E tests from: ${PWD}"

SRC_DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"

${SRC_DIR}/cleanup.sh
${SRC_DIR}/setup-kubeflex.sh
${SRC_DIR}/manage-type-k8s.sh
${SRC_DIR}/test-controller-image-update.sh
${SRC_DIR}/manage-type-vcluster.sh
${SRC_DIR}/manage-type-external.sh
${SRC_DIR}/manage-ctx.sh
${SRC_DIR}/test-postcreate-completion.sh -t k8s
${SRC_DIR}/test-postcreate-completion.sh -t vcluster
${SRC_DIR}/test-postcreatehook-retry.sh -t k8s
${SRC_DIR}/test-postcreatehook-retry.sh -t vcluster
${SRC_DIR}/test-custom-cluster-name.sh

echo "SUCCESS: ALL TESTS PASSED..."
