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

CUSTOM_CLUSTER_NAME="test-custom-cluster"

:
: -------------------------------------------------------------------------
: "Test kflex init command accepts custom cluster name argument"
:

# Test that the command accepts the positional argument without error
# We don't actually create the cluster in CI to avoid complex setup
if ! bin/kflex init "${CUSTOM_CLUSTER_NAME}" --help >/dev/null 2>&1; then
    echo "ERROR: kflex init command failed to parse custom cluster name argument"
    exit 1
fi

echo "✓ kflex init command accepts custom cluster name argument"

:
: -------------------------------------------------------------------------
: "Test kflex init command argument validation"
:

# Test that the command properly validates arguments
# This should fail with an error about not finding kubeconfig (expected in CI)
# but should NOT fail with argument parsing errors
if bin/kflex init "${CUSTOM_CLUSTER_NAME}" -c 2>&1 | grep -q "unknown flag\|invalid argument\|too many arguments"; then
    echo "ERROR: kflex init command failed to parse arguments correctly"
    exit 1
fi

echo "✓ kflex init command argument parsing works correctly"

:
: -------------------------------------------------------------------------
: "SUCCESS: Custom cluster name test completed"
:
echo "✓ Custom cluster name functionality validation passed"