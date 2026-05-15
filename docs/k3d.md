# Installing and Using KubeFlex on k3d

[k3d](https://k3d.io) is a lightweight wrapper that runs [k3s](https://k3s.io)
(a minimal Kubernetes distribution by Rancher) inside Docker containers.
It is ideal for local development and testing environments where a lightweight,
fast-starting Kubernetes cluster is preferred over a full-featured distribution.

KubeFlex supports k3d as a hosting cluster, making it a great option for developers
who want a resource-efficient alternative to kind.

## When to Use k3d with KubeFlex

Consider using k3d as your KubeFlex hosting cluster when:

- You want a **lightweight local development environment** with lower memory usage than kind.
- You are working on a **resource-constrained machine** (e.g., a laptop with limited RAM).
- You need **faster cluster startup times** for iterative development.
- You are exploring **edge computing scenarios** where k3s's minimal footprint is an advantage.

## Prerequisites

Before installing KubeFlex on k3d, ensure you have the following installed:

- [Docker](https://docs.docker.com/get-docker/) (running)
- [k3d](https://k3d.io/#installation) v5.6.0 or later
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [helm](https://helm.sh/docs/intro/install/)
- KubeFlex CLI (`kflex`) — see the [installation guide](./users.md#installation)

### Installing k3d

```shell
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
k3d version
```

## Setting Up a k3d Cluster for KubeFlex

These steps have been tested with k3d v5.6.0.

KubeFlex requires nginx ingress with SSL passthrough enabled. Since k3d uses
Traefik as its default ingress controller, you must disable Traefik and install
nginx ingress manually.

**Step 1: Create a k3d cluster with Traefik disabled and nginx ingress:**

```shell
k3d cluster create -p "9443:443@loadbalancer" \
  --k3s-arg "--disable=traefik@server:*" kubeflex
```

**Step 2: Install nginx ingress with SSL passthrough enabled:**

```shell
helm install ingress-nginx ingress-nginx \
  --set "controller.extraArgs.enable-ssl-passthrough=true" \
  --repo https://kubernetes.github.io/ingress-nginx \
  --version 4.6.1 \
  --namespace ingress-nginx \
  --create-namespace
```

**Step 3: Install KubeFlex on the k3d cluster:**

```shell
kflex init --hostContainerName k3d-kubeflex-server-0
```

The `--hostContainerName` flag tells KubeFlex the name of the k3d server
container, which differs from a kind cluster setup.

**NOTE**: After running `kflex init`, you **MUST NOT** change your kubeconfig
current context by any other means before using `kflex create`.

## Creating a Control Plane on k3d

Once KubeFlex is installed, creating control planes on k3d works the same as
on any other hosting cluster. For example, to create a control plane of type `k8s`:

```shell
kflex create cp1
```

To create a control plane of type `vcluster`:

```shell
kflex create cp2 --type vcluster
```

To create a control plane of type `ocm`:

```shell
kflex create cp3 --type ocm
```

Refer to the [User's Guide](./users.md#control-plane-types) for the full list
of supported control plane types and their capabilities.

## Capabilities and Features

When running KubeFlex on k3d, all standard KubeFlex control plane types are supported:

| Control Plane Type | Supported on k3d |
|--------------------|-----------------|
| `k8s`              | ✅ Yes           |
| `vcluster`         | ✅ Yes           |
| `ocm`              | ✅ Yes           |
| `host`             | ✅ Yes           |

Post-create hooks, context switching with `kflex ctx`, and kubeconfig management
all work identically to a kind-based setup.

## Limitations

- These instructions have been tested with **k3d v5.6.0** only. Compatibility with other versions is not guaranteed.
- The `--hostContainerName` flag must match your actual k3d server container name. If you used a different cluster name than `kubeflex`, adjust accordingly (e.g., `k3d-<your-cluster-name>-server-0`).
- k3d runs k3s inside Docker, so Docker must be running at all times while using your KubeFlex environment.

## Verifying the Installation

After setup, verify your control planes are running:

```shell
kubectl get controlplanes
```

Expected output:
NAME   SYNCED   READY   AGE
cp1    True     True    2m

## Cleaning Up

To delete your k3d cluster and all KubeFlex control planes:

```shell
k3d cluster delete kubeflex
```

## References

- [k3d Documentation](https://k3d.io)
- [k3s Documentation](https://k3s.io)
- [KubeFlex User's Guide](./users.md)
- [KubeFlex Architecture](./architecture.md)
