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

# Verify that a locally built kflex reports a version of the form
# <release tag>+<short commit>.

set -e

cd "$(dirname "$0")/.."

make build

expected_commit=$(git rev-parse --short HEAD)

# kflex version exits nonzero when no cluster is reachable; the version line is printed first
actual_version=$(bin/kflex version | awk '/^Kubeflex version:/ {print $3; exit}')

if [[ ! "$actual_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?\+${expected_commit}$ ]]; then
    echo "built binary reports version '${actual_version}', expected a release tag followed by +${expected_commit}"
    exit 1
fi

echo "built binary reports version ${actual_version}"
