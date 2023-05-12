# kubeflex
A flexible and scalable solution for running Kubernetes control plane APIs.




## Hacks

### How to open a psql command in-cluster

```shell
kubectl run -i --tty --rm debug -n vks-system --image=postgres --restart=Never -- bash
psql -h mypsql-postgresql.vks-system.svc -U postgres
```

### How to create a secret with all certs

```shell
kubectl create secret generic k8s-certs-test --from-file=apiserver-kubelet-client.crt=/path/to/.ssh/id_rsa