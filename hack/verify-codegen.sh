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

# This script verifies that the generated code is up-to-date.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}"

# Create temporary directory for generated code
DIFFROOT="${REPO_ROOT}/pkg/generated"
TMP_DIFFROOT="$(mktemp -d)"
trap 'rm -rf "${TMP_DIFFROOT}"' EXIT

# Copy current generated code to temp
if [[ -d "${DIFFROOT}" ]]; then
    cp -a "${DIFFROOT}/." "${TMP_DIFFROOT}"
fi

# Delete entire generated code tree for reproducibility
echo "Clearing pkg/generated..."
rm -rf "${REPO_ROOT}/pkg/generated"

# Invoke update-codegen.sh to regenerate code
echo "Regenerating code..."
"${REPO_ROOT}/hack/update-codegen.sh"

echo "Verifying generated code is up-to-date..."

# Check for differences
if ! diff -Naupr "${TMP_DIFFROOT}" "${DIFFROOT}"; then
    echo ""
    echo "Generated code is out of date. Please run:"
    echo "    hack/update-codegen.sh"
    echo ""
    exit 1
fi

echo "Generated code is up-to-date."
