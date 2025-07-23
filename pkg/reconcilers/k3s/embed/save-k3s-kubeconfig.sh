#!/bin/bash -u
# Envs: DNS_SVC DNS_INGRESS
set -e
# Install packages and setup envs
apk add --no-cache jq yq curl base64
APISERVER=https://kubernetes.default.svc
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt
KUBECONFIG=/etc/rancher/k3s/config.yaml
# Process kubeconfig
cp /etc/rancher/k3s/k3s.yaml $KUBECONFIG
yq e -i ".clusters[0].cluster.server = \"${DNS_INGRESS}\"" $KUBECONFIG
dataConfig=$(cat $KUBECONFIG | yq -r | base64 | tr -d '[:space:]"')
yq e -i ".clusters[0].cluster.server = \"${DNS_SVC}\"" $KUBECONFIG
dataConfigIncluster=$(cat $KUBECONFIG | yq -r | base64 | tr -d '[:space:]"')
dataJson="{\"data\": {\"config\": \"${dataConfig}\", \"config-incluster\": \"${dataConfigIncluster}\"}}"
# Validation and formatting before payload submission
dataJson=$(echo $dataJson | jq -c) 
curl --cacert ${CACERT} --header "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/merge-patch+json" -X PATCH -d "${dataJson}" ${APISERVER}/api/v1/namespaces/${NAMESPACE}/secrets/k3s-config
