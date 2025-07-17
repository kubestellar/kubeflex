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
# set -e # we do not want to exit on error



:
: -------------------------------------------------------------------------
: Create a simple Control plane
:
CTX_NAME="check-ctx-list"
NEW_CTX_NAME="check-ctx-list-renamed"

./bin/kflex create $CTX_NAME --chatty-status=false

# -------------------------------------------------------------------------
# Validate kflex ctx list output for managed context
# -------------------------------------------------------------------------
CTX_LIST_OUTPUT=$(./bin/kflex ctx list)
echo "$CTX_LIST_OUTPUT"

# Check for header columns
if [[ ! -z "$(echo "$CTX_LIST_OUTPUT" | grep -q "CONTEXT" || ! echo "$CTX_LIST_OUTPUT" | grep -q "MANAGED BY KFLEX" || ! echo "$CTX_LIST_OUTPUT" | grep -q "CONTROLPLANE")" ]]; then
  echo "ERROR: kflex ctx list output missing expected columns" >&2
  exit 1
fi

# Check for managed context row
if [[ ! -z "$(echo "$CTX_LIST_OUTPUT" | grep -q "${CTX_NAME}" || ! echo "$CTX_LIST_OUTPUT" | grep -q "yes" || ! echo "$CTX_LIST_OUTPUT" | grep -q "${CTX_NAME}")" ]]; then
  echo "ERROR: kflex ctx list output missing managed context info" >&2
  exit 1
fi

:
: -------------------------------------------------------------------------
: Check that $CTX_NAME context, user and cluster entries are present
:
[[ $(kubectl config get-contexts -o=name | grep -c "$CTX_NAME") -ne 1 ]] && exit 1
[[ $(kubectl config get-users | grep -c "$CTX_NAME") -ne 1 ]] && exit 1
[[ $(kubectl config get-clusters | grep -c "$CTX_NAME") -ne 1 ]] && exit 1

:
: -------------------------------------------------------------------------
: Rename context ${CTX_NAME} to ${NEW_CTX_NAME}
:
./bin/kflex ctx rename ${CTX_NAME} ${NEW_CTX_NAME}

:
: -------------------------------------------------------------------------
: Check that $CTX_NAME context, user and cluster entries are deleted
: and verify $NEW_CTX_NAME entries are inserted
:
[[ $(kubectl config get-contexts -o=name | grep -c "$NEW_CTX_NAME") -ne 1 ]] && exit 1
[[ $(kubectl config get-users | grep -c "$NEW_CTX_NAME") -ne 1 ]] && exit 1
[[ $(kubectl config get-clusters | grep -c "$NEW_CTX_NAME") -ne 1 ]] && exit 1
[[ $(kubectl config get-contexts -o=name | grep -c "$CTX_NAME") -ne 0 ]] && exit 1
[[ $(kubectl config get-users | grep -c "$CTX_NAME") -ne 0 ]] && exit 1
[[ $(kubectl config get-clusters | grep -c "$CTX_NAME") -ne 0 ]] && exit 1
echo "TEST: kflex ctx rename PASSED"

:
: -------------------------------------------------------------------------
: Delete context $NEW_CTX_NAME and verify it is removed from kubeconfig file
:
./bin/kflex ctx delete ${NEW_CTX_NAME}

:
: -------------------------------------------------------------------------
: Check $NEW_CTX_NAME is removed from kubeconfig file
:
[[ $(kubectl config get-contexts -o=name | grep -c "$NEW_CTX_NAME") -ne 0 ]] && exit 1
[[ $(kubectl config get-users | grep -c "$NEW_CTX_NAME") -ne 0 ]] && exit 1
[[ $(kubectl config get-clusters | grep -c "$NEW_CTX_NAME") -ne 0 ]] && exit 1
echo "TEST: kflex ctx delete PASSED"