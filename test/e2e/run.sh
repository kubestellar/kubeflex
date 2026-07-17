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
cluster_type=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release)
      release="$2"
      shift 2
      ;;
    --cluster-type)
      cluster_type="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

cluster_type=${cluster_type:-kind}
host_context="${cluster_type}-kubeflex"

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
${SRC_DIR}/setup-kubeflex.sh --release "${release}" --cluster-type "$cluster_type"

if [ -z "${release}" ]; then
    ${SRC_DIR}/test-informers.sh
else
    : there is no local watch-objs command, neither source nor executable, to use
fi

${SRC_DIR}/manage-type-k8s.sh --host-context "$host_context"

if [ -z "${release}" ]; then
    # This test is only appropriate when testing the local copy
    ${SRC_DIR}/test-controller-image-update.sh --cluster-type "$cluster_type"
fi

${SRC_DIR}/manage-type-vcluster.sh --host-context "$host_context"

case "$cluster_type" in
    (kind) ${SRC_DIR}/manage-type-external.sh ;;
    (k3d)
        echo -e "\n================================"
        echo "External cluster adoption is not working when the host cluster is made by k3d"
        echo -e "================================\n"
        ;;
    (*) echo "Unexpected cluster_type '$cluster_type' !" >&2
        exit 1 ;;
esac

${SRC_DIR}/manage-ctx.sh --host-context "$host_context"
${SRC_DIR}/test-postcreate-completion.sh -t k8s
${SRC_DIR}/test-postcreate-completion.sh -t vcluster
${SRC_DIR}/test-postcreatehook-retry.sh -t k8s
${SRC_DIR}/test-postcreatehook-retry.sh -t vcluster
${SRC_DIR}/test-custom-cluster-name.sh

echo "SUCCESS: ALL TESTS PASSED..."
