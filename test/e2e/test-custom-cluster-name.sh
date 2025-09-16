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

SRC_DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"
source "${SRC_DIR}/setup-shell.sh"

set -x # echo so that users can understand what is happening
set -e # exit on error

CUSTOM_CLUSTER_NAME="test-custom-cluster"

:
: -------------------------------------------------------------------------
: "Create kubeflex cluster with custom name"
:
: Ensure any existing default kind cluster is removed to avoid port conflicts
kind delete cluster --name kubeflex || true
kind delete cluster --name ext || true

: Stop any containers binding to host ports 9080 or 9443 if present
if command -v docker >/dev/null 2>&1; then
    for PORT in 9080 9443; do
        CIDS=$(docker ps --format '{{.ID}} {{.Ports}}' | grep -E ":${PORT}->" | awk '{print $1}' || true)
        if [ -n "$CIDS" ]; then
            echo "Stopping containers using port $PORT: $CIDS"
            echo "$CIDS" | xargs docker stop || true
            echo "$CIDS" | xargs docker rm || true
        fi
    done
    : Clean up any existing kind network
    docker network rm kind || true
fi
sleep 5

if ! bin/kflex init -c "${CUSTOM_CLUSTER_NAME}"; then
    echo "ERROR: Failed to create kubeflex cluster with custom name"
    exit 1
fi

:
: -------------------------------------------------------------------------
: "Verify cluster was created with correct name"
:
if [ -z "$(kind get kubeconfig --name "${CUSTOM_CLUSTER_NAME}" 2>/dev/null)" ]; then
    echo "ERROR: Custom cluster '${CUSTOM_CLUSTER_NAME}' was not created"
    exit 1
fi
echo "✓ Cluster '${CUSTOM_CLUSTER_NAME}' was created successfully"

:
: -------------------------------------------------------------------------
: "Verify kubeconfig context was set correctly"
:
EXPECTED_CONTEXT="kind-${CUSTOM_CLUSTER_NAME}"
if [ -z "$(kubectl config get-contexts -o name | grep "^${EXPECTED_CONTEXT}$")" ]; then
    echo "ERROR: Expected kubeconfig context '${EXPECTED_CONTEXT}' was not found"
    exit 1
fi
echo "✓ Kubeconfig context '${EXPECTED_CONTEXT}' was created successfully"

:
: -------------------------------------------------------------------------
: "Verify cluster is accessible and functional"
:
kubectl --context "${EXPECTED_CONTEXT}" cluster-info

:
: -------------------------------------------------------------------------
: "Cleanup: Delete the test cluster"
:
kind delete cluster --name "${CUSTOM_CLUSTER_NAME}"

:
: -------------------------------------------------------------------------
: "SUCCESS: Custom cluster name test completed"
:
echo "✓ Custom cluster name functionality works correctly"