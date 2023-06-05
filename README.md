# kubeflex

A flexible and scalable solution for running Kubernetes control plane APIs.

## Goals

- Provide lightweight Kube API Server instances and selected controllers as a service.
- Provide a flexible architecture for the storage backend, e.g.:
    - shared DB for API servers
    - dedicated DB for each API server
    - etcd DB or Kine + Postgres DB
- Flexibility in choice of API Server build:
    - upstream Kube (e.g. `registry.k8s.io/kube-apiserver:v1.27.1`)    
    - trimmed down builds (e.g. [multicluster control plane](https://github.com/open-cluster-management-io/multicluster-controlplane))
- Single binary CLI for improved user experience:
    - initialize, install operator, manage lifecycle of control planes and contexts

## Architecture

![image info](./docs/images/kubeflex-arch.png)

## Prereqs

- go version>=go1.19.2
- git
- make
- docker
- kind

## Quickstart

Clone this repo, build the binaries and add the binary to your path:

```shell
git clone git@github.ibm.com:dettori/kubeflex.git
cd kubeflex
make build-all
export PATH=$PATH:$(pwd)/bin
```

Create kind cluster with ingress and install kubeflex controller:

```shell
kflex init --create-kind
```

Create a control pkane

```shell 
kflex create cp1
```

Interact with the new control plane, for example get namespaces and create a new one.

```shell
kubectl get ns
kubectl create ns myns
```

Create a second control plane and check that you don't have the namespace created in the
first one:

```shell
kflex create cp2
kubectl get ns
```

To go back to the hosting cluster context, use the `ctx` command:

```shell
kfkex ctx
```

To switch back to a control plane context, use the 
`create <control plane name>` command, e.g:

```shell
kfkex ctx cp1
```

To delete a control plane, use the `delete <control plane name>` command, e.g:

```shell
kflex delete cp1
```


## Useful Debugging Hacks

### How to open a psql command in-cluster

```shell
kubectl run -i --tty --rm debug -n vks-system --image=postgres --restart=Never -- bash
psql -h mypsql-postgresql.vks-system.svc -U postgres
```

### How to create a secret with all certs

```shell
kubectl create secret generic k8s-certs-test --from-file=apiserver-kubelet-client.crt=/path/to/.ssh/id_rsa
```

### How to view certs info

```shell
openssl x509 -noout -text -in certs/apiserver.crt 
```

### Get decoded value from secret

```shell
kubectl get secret -n cp3-system admin-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d
```

### How to attach a ephemeral container to debug

```shell
kubectl debug -n cp1-system -it kube-controller-manager-676c565f96-r952b --image=busybox:1.28 --target=kube-controller-manager
```

### Getting all the command args for a process

```
cat /proc/<pid>/cmdline | sed -e "s/\x00/ /g"; echo
```

### Manually install the postgres helm chart

```
helm install postgres oci://registry-1.docker.io/bitnamicharts/postgresql --set primary.extendedConfiguration="max_connections = 1000" -n kflex-system
```