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



# :
# : -------------------------------------------------------------------------
# : Create a ControlPlane of type vcluster
# :
# OLD_CTX_NAME="testcp"
# NEW_CTX_NAME="testcp-renamed"
# pwd
# ./bin/kflex create $OLD_CTX_NAME --chatty-status=false

# :
# : -------------------------------------------------------------------------
# : Rename context ${OLD_CTX_NAME} to ${NEW_CTX_NAME}
# :
# ./bin/kflex ctx
# ./bin/kflex ctx rename ${OLD_CTX_NAME} ${NEW_CTX_NAME}

# :
# : -------------------------------------------------------------------------
# : Verify if testcp is removed and testcp-renamed is present
# :
# _has_new_ctx=0
# _has_old_ctx=0
# for ctx in $(kubectl config get-contexts -o=name)
# do
#     echo "ctx: $ctx"
#     [[ $ctx == "${NEW_CTX_NAME}" ]] && _has_new_ctx=1 
#     [[ $ctx == "${OLD_CTX_NAME}" ]] && _has_old_ctx=1 
# done


# :
# : -------------------------------------------------------------------------
# : Delete ControlPlane testcp
# : SUCCESS: new context replaces old one
# : FAILURE: old context still present
# if [[ $_has_new_ctx -eq 1 && $_has_old_ctx -eq 0 ]]; then
#     echo "SUCCESS"
#     ./bin/kflex delete ${NEW_CTX_NAME}  --chatty-status=false
#     exit 0
# else 
#     echo "FAILURE"
#     exit 1
# fi

# :
# : -------------------------------------------------------------------------
# : Delete ControlPlane testcp for cleanuo

# ./bin/kflex delete ${OLD_CTX_NAME}  --chatty-status=false
