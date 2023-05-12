#!/bin/bash

create_or_update_certs_secret() {
    kubectl delete -n ${VKS_NS} secret k8s-certs &> /dev/null
    kubectl create -n ${VKS_NS} secret generic k8s-certs \
    --from-file=${VKS_HOME}/ca.crt \
    --from-file=${VKS_HOME}/ca.key \
    --from-file=${VKS_HOME}/apiserver-kubelet-client.crt \
    --from-file=${VKS_HOME}/apiserver-kubelet-client.key \
    --from-file=${VKS_HOME}/front-proxy-client.crt \
    --from-file=${VKS_HOME}/front-proxy-client.key \
    --from-file=${VKS_HOME}/front-proxy-ca.crt \
    --from-file=${VKS_HOME}/sa.pub \
    --from-file=${VKS_HOME}/sa.key \
    --from-file=${VKS_HOME}/apiserver.crt \
    --from-file=${VKS_HOME}/apiserver.key
}  

VKS_NS=cp1-system
VKS_HOME=certs

ls certs
create_or_update_certs_secret

