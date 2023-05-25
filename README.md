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

### How to view certs info

```shell
openssl x509 -noout -text -in certs/apiserver.crt 
```

### Get decoded value from secret

```shell
kubectl get secret -n vks1-system k8s-certs -o jsonpath='{.data.ca\.crt}' | base64 -d
kubectl get secret -n cp3-system admin-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d
```

### How to attach a ephemeral container to debug

Example:

```shell
kubectl debug -n cp1-system -it kube-controller-manager-676c565f96-r952b --image=busybox:1.28 --target=kube-controller-manager

kubectl debug -n cp3-system -it kube-apiserver-c4cf49b88-pfglt --image=busybox:1.28 --target=kube-apiserver
```

### Using domain names in /etc/hosts

You need either to add the entries in /etc/hosts (no wildcards can be used) or use something like
(dnsmask)[https://www.larry.dev/no-more-etc-hosts-on-mac-with-dnsmasq/sudo]

### Getting the all the command args for a process

```
cat /proc/<pid>/cmdline | sed -e "s/\x00/ /g"; echo
```