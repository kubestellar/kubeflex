set -e
apk add --no-cache jq curl base64 openssl

APISERVER=https://kubernetes.default.svc
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace)
TOKEN=$(cat ${SERVICEACCOUNT}/token)
CACERT=${SERVICEACCOUNT}/ca.crt
# curl --cacert ${CACERT} --header "Authorization: Bearer ${TOKEN}" -X GET ${APISERVER}/api/v1/namespaces/${NAMESPACE}/configmaps

mkdir -p /var/lib/rancher/k3s/server/tls
curl -sL https://raw.githubusercontent.com/k3s-io/k3s/refs/tags/v1.30.13%2Bk3s1/contrib/util/generate-custom-ca-certs.sh | bash -
# Build configmap data
echo "building json payload"

data_value_string="{\"data\":{"
while read -r f;
do
  data_value_string+="\"$f\": \"$(base64 $f)\","
  echo "filename=$f"
done < <(find /var/lib/rancher/k3s/server/tls -type f -maxdepth 1)
while read -r f;
do
  data_value_string+="\"etcd-$f\": \"$(base64 $f)\","
  echo "filename=$f"
done < <(find /var/lib/rancher/k3s/server/tls/etcd -type f -maxdepth 1)

# remove last comma to validate json format
data_value_string=${data_value_string%?}
data_value_string+="}}"
# validate json payload
#echo $data_value_string | jq empty
data_value_string=$(echo $data_value_string | jq -c)

curl --cacert ${CACERT} --header "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/merge-patch+json" -X PATCH -d "${data_value_string}" ${APISERVER}/api/v1/namespaces/${NAMESPACE}/configmaps/k3s-certs
