#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

create_kind_instance(){
cat <<EOF | kind create cluster --name kubeflex --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF
}

install_and_patch_nginx_ingress() {
# Note: The nginx ingress manifests contains kind specific patches to forward 
# the hostPorts to the ingress controller, set taint tolerations 
# and schedule it to the custom labelled node.
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

cat <<EOF > /tmp/nginx-controller-patch.yaml
spec:
  template:
    spec:
      containers:
        - name: controller
          args:
            - /nginx-ingress-controller
            - --election-id=ingress-nginx-leader
            - --controller-class=k8s.io/ingress-nginx
            - --ingress-class=nginx
            - --configmap=\$(POD_NAMESPACE)/ingress-nginx-controller
            - --validating-webhook=:8443
            - --validating-webhook-certificate=/usr/local/certificates/cert
            - --validating-webhook-key=/usr/local/certificates/key
            - --watch-ingress-without-class=true
            - --publish-status-address=localhost
            - --enable-ssl-passthrough
EOF

kubectl -n ingress-nginx patch deployment/ingress-nginx-controller \
--patch-file /tmp/nginx-controller-patch.yaml
}

#######################################################################

create_kind_instance

install_and_patch_nginx_ingress