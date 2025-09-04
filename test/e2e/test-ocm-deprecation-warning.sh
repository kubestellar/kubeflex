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

OCM_CP_NAME="cp-ocm-test"
K8S_CP_NAME="cp-k8s-test"

cleanup() {
    echo "Cleaning up test resources..."
    kubectl delete controlplane "${OCM_CP_NAME}" --ignore-not-found=true
    kubectl delete controlplane "${K8S_CP_NAME}" --ignore-not-found=true
}

trap cleanup EXIT

:
: -------------------------------------------------------------------------
: Clean up any existing test resources
:
echo "Cleaning up any existing test resources..."
cleanup

:
: -------------------------------------------------------------------------
: Test case: OCM control plane creation should display deprecation warning
:
echo "Testing OCM deprecation warning display..."

output=$(./bin/kflex create "${OCM_CP_NAME}" --type ocm --chatty-status=false 2>&1 || true)

if echo "$output" | grep -q "WARNING: OCM-type control plane is deprecated"; then
    echo "SUCCESS: OCM deprecation warning was displayed"
else
    echo "ERROR: OCM deprecation warning was not displayed"
    echo "Command output:"
    echo "$output"
    exit 1
fi

:
: -------------------------------------------------------------------------
: Test case: Non-OCM control planes should NOT display deprecation warning
:
echo "Testing that non-OCM control planes don't show deprecation warning..."

output=$(./bin/kflex create "${K8S_CP_NAME}" --type k8s --chatty-status=false 2>&1 || true)

if echo "$output" | grep -q "WARNING: OCM-type control plane is deprecated"; then
    echo "ERROR: OCM deprecation warning was incorrectly displayed for k8s control plane"
    echo "Command output:"
    echo "$output"
    exit 1
else
    echo "SUCCESS: OCM deprecation warning was not displayed for k8s control plane"
fi

:
: -------------------------------------------------------------------------
: SUCCESS: OCM deprecation warning test completed
:
echo "Test completed successfully"
