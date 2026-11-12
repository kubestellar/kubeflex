#!/usr/bin/env bash

# Copyright 2026 The KubeStellar Authors.
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

# Usage: hack/k3d-import-image.sh IMAGE CLUSTER
#
# This script imports a container image from the local Docker daemon into the
# nodes of the given k3d cluster.
#
# A plain `k3d image import` occasionally fails to import the image and still
# exits with status 0 (https://github.com/k3d-io/k3d/issues/1484), after which
# Pods using the image are stuck in ImagePullBackOff
# (https://github.com/kubestellar/kubeflex/issues/699). That happens in k3d's
# default `tools-node` mode, which saves the image to a tarball in a shared
# volume and then imports the tarball into each node; the import sometimes
# fails with "no such file or directory", and k3d only logs that.
# The `direct` mode streams the image into each node instead. It can fail
# intermittently too (https://github.com/k3d-io/k3d/issues/900), but then k3d
# exits with a non-zero status. So this script uses `direct` mode and retries
# a failed import.

set -o errexit
set -o nounset
set -o pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 IMAGE CLUSTER" >&2
    exit 1
fi

image=$1
cluster=$2
max_attempts=3

for attempt in $(seq 1 "$max_attempts"); do
    if k3d image import --mode direct "$image" --cluster "$cluster"; then
        exit 0
    fi
    echo "$0: attempt $attempt of $max_attempts to import $image into k3d cluster $cluster failed" >&2
done
exit 1
