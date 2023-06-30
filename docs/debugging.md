# Debugging Kubeflex

## Useful Debugging Hacks

### How to open a psql command in-cluster

```shell
kubectl run -i --tty --rm debug -n kubeflex-system --image=postgres --restart=Never -- bash
psql -h mypsql-postgresql.kubeflex-system.svc -U postgres
```

### How to view certs info

```shell
openssl x509 -noout -text -in certs/apiserver.crt 
```

### Get decoded value from secret

```shell
NAMESPACE= # your namespace
NAME= # your secret name
kubectl get secrets -n ${NAMESPACE} ${NAME} -o jsonpath='{.data.apiserver\.crt}' | base64 -d
```

### How to attach a ephemeral container to debug

```shell
NAMESPACE= # your namespace
NAME= # pod name
CONTAINER= # container name
IMAGE=ubuntu:latest
kubectl debug -n ${NAMESPACE} -it ${NAME} --image=${IMAGE} --target=${CONTAINER} -- bash
# e.g. kubectl debug -n ${NAMESPACE} -it ${NAME} --image=curlimages/curl:8.1.2 --target=${CONTAINER} -- sh
```

### Getting all the command args for a process

```shell
cat /proc/<pid>/cmdline | sed -e "s/\x00/ /g"; echo
```

### Install the kubeflex operator with the OCI helm chart

```shell
helm install test -n kubeflex-system --create-namespace oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator --version v0.1.0
```

### How to communicate between kind clusters on the same node

One approach that is independent of local machine IP is to use the internal DNS address of
docker containers. The address is the name of the docker container. For kubflex that
address is `kubeflex-control-plane`. For example, if I have a nodeport service on 
`kubeflex-control-plane` with port 30080 and I want to access it from another kind cluster
the internal address to use is `https://kubeflex-control-plane:30080`

### Commands to package and push a chart to a OCI registry

```shell
helm package charts/multicluster-controlplane
helm push multicluster-controlplane-chart-0.1.0.tgz oci://quay.io/pdettori
```