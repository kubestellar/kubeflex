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
kubectl debug -n ${NAMESPACE} -it ${NAME} --image=busybox:1.28 --target=${CONTAINER}
```

### Getting all the command args for a process

```shell
cat /proc/<pid>/cmdline | sed -e "s/\x00/ /g"; echo
```

### Install the kubeflex operator with the OCI helm chart

```shell
helm install test -n kubeflex-system --create-namespace oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator --version v0.1.0
```

### Installing CLI as brew tap

```shell
brew tap kubestellar/kubeflex https://github.com/kubestellar/kubeflex
brew install kubeflex
```