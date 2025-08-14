#!/bin/bash -u
# Envs: K3S_DATA_DIR K3S_CONTROLPLANE_SECRET_NAME
set -e
[[ -z $K3S_DATA_DIR || -z $K3S_CONTROLPLANE_SECRET_NAME ]] && exit -1
# Install packages and setup envs
apk add --no-cache jq yq curl base64
APISERVER=https://kubernetes.default.svc
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt
# Process kubeconfig
k3sToken=$(cat $K3S_DATA_DIR/server/token | base64)
dataJson="{\"data\": {\"token\": \"${k3sToken}\"}}"
# Validation and formatting before payload submission
dataJson=$(echo $dataJson | jq -c) 
curl --cacert ${CACERT} --header "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/merge-patch+json" -X PATCH -d "${dataJson}" ${APISERVER}/api/v1/namespaces/${NAMESPACE}/secrets/${K3S_CONTROLPLANE_SECRET_NAME}
