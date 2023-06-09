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

```
cat /proc/<pid>/cmdline | sed -e "s/\x00/ /g"; echo
```
