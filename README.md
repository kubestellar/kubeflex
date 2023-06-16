# <img alt="Logo" width="90px" src="./docs/images/kubeflex-logo.png" style="vertical-align: middle;" />  KubeFlex

A flexible and scalable platform for running Kubernetes control plane APIs.

## Goals

- Provide lightweight Kube API Server instances and selected controllers as a service.
- Provide a flexible architecture for the storage backend, e.g.:
    - shared DB for API servers,
    - dedicated DB for each API server,
    - etcd DB or Kine + Postgres DB
- Flexibility in choice of API Server build:
    - upstream Kube (e.g. `registry.k8s.io/kube-apiserver:v1.27.1`),    
    - trimmed down API Server builds (e.g. [multicluster control plane](https://github.com/open-cluster-management-io/multicluster-controlplane))
- Single binary CLI for improved user experience:
    - initialize, install operator, manage lifecycle of control planes and contexts.

## Installation

[kind](https://kind.sigs.k8s.io) is required. Note that we plan to add support 
for other Kube distros. Note that a hosting kind cluster is created automatically 
by the kubeflex CLI.

Download the latest kubeflex CLI binary release for your OS/Architecture from the 
[release page](https://github.com/kubestellar/kubeflex/releases) and copy it
to `/usr/local/bin` or another location in your `$PATH`.

If you have [Homebrew](https://brew.sh), use the following commands to install kubeflex:

```shell
brew tap kubestellar/kubeflex https://github.com/kubestellar/kubeflex
brew install kubeflex
```

## Quickstart

Create the hosting kind cluster with ingress controller and install 
the kubeflex operator:

```shell
kflex init --create-kind
```

Create a control plane:

```shell 
kflex create cp1
```

Interact with the new control plane, for example get namespaces and create a new one:

```shell
kubectl get ns
kubectl create ns myns
```

Create a second control plane and check that the namespace created in the
first control plane is not present:

```shell
kflex create cp2
kubectl get ns
```

To go back to the hosting cluster context, use the `ctx` command:

```shell
kflex ctx
```

To switch back to a control plane context, use the 
`create <control plane name>` command, e.g:

```shell
kflex ctx cp1
```

To delete a control plane, use the `delete <control plane name>` command, e.g:

```shell
kflex delete cp1
```

## Architecture

![image info](./docs/images/kubeflex-arch.png)