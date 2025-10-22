# KubeFlex Architecture

KubeFlex implements a sophisticated multi-tenant architecture that separates control plane management from workload execution. This page details the components, control plane lifecycle, storage backends, networking, and automation hooks.

## High-Level Diagram

![KubeFlex Architecture](../../../docs/images/kubeflex-architecture.png)

## Core Components

### KubeFlex Controller (Operator)

The KubeFlex controller is the central operator that manages the lifecycle of control planes in the hosting cluster. It continuously reconciles `ControlPlane` custom resources to ensure the desired state matches the actual state. The controller is responsible for creating and managing namespaces, API servers, controller managers, Services, Ingress or Routes, and kubeconfig secrets for each tenant control plane. Additionally, it orchestrates PostCreateHooks and maintains status reporting to provide visibility into the health and state of each control plane.

### Tenant Control Planes

Each tenant receives a dedicated, isolated API server that provides strong multi-tenant isolation. Every tenant control plane includes a controller manager that runs essential Kubernetes controllers such as namespace, garbage collection, and service account controllers. This architecture ensures that each tenant has their own control plane without the overhead of running a complete, separate Kubernetes cluster.

### Flexible Data Plane

KubeFlex supports multiple data plane configurations to meet different requirements. Workloads can run on shared hosting cluster nodes, leverage vCluster virtual nodes for additional isolation, or use dedicated KubeVirt virtual machines as an integration point for stronger compute isolation.

### CLI (kflex)

The `kflex` command-line interface provides a unified tool for managing KubeFlex installations and control planes. It can initialize the hosting cluster, optionally creating a kind cluster and installing the operator. The CLI handles creating, listing, and deleting control planes, and manages kubeconfig contexts through the `ctx` command to enable seamless switching between different control planes.

### Storage Abstraction

KubeFlex provides flexible storage backends depending on the control plane type. For `k8s` control planes, a shared PostgreSQL instance accessed via Kine provides efficient, multi-tenant storage. For `vcluster` control planes, each instance uses embedded sqlite or etcd with a persistent volume for data durability. Note that OCM functionality is now provided by running OCM inside `vcluster` control planes, as the standalone OCM control plane type has been deprecated.

## Supported Control Plane Types

KubeFlex supports multiple control plane types to accommodate different use cases and resource requirements.

### k8s

The `k8s` control plane type provides an upstream Kubernetes API server with a subset of core controllers. This type does not support running pod workloads directly in the control plane and typically uses around 350MB of memory per instance. It uses a shared PostgreSQL backend via Kine for efficient multi-tenant storage.

### k3s

The `k3s` control plane type leverages the [K3s distribution](https://k3s.io), which packages Kubernetes as a single binary with embedded etcd and components. In KubeFlex, k3s runs as a StatefulSet with dedicated persistent volumes for data storage. It is optimized for edge deployments and resource-constrained environments, offering a lightweight yet complete Kubernetes cluster while maintaining full Kubernetes compatibility.

### vcluster

The `vcluster` control plane type provides a virtual Kubernetes control plane that runs inside a namespace of the KubeFlex hosting cluster. Based on the [vCluster project](https://www.vcluster.com), it includes a virtual API server and embedded etcd that run together in a single process, with a persistent volume mounted for data persistence. Vcluster uses a syncer component to mirror workload resources (pods, services, configmaps, etc.) from the virtual control plane to the hosting cluster, where they are actually scheduled and executed on the host's worker nodes. This virtualization approach enables strong control plane isolation while sharing the underlying data plane infrastructure. OCM functionality is supported by running OCM inside vcluster control planes rather than as a standalone control plane type.

### host

The `host` control plane type exposes the hosting cluster itself as a control plane abstraction. This type only provides in-cluster kubeconfig access since it represents the cluster where KubeFlex is running.

### external

The `external` control plane type represents an existing Kubernetes cluster not created by KubeFlex. It is adopted into KubeFlex management via a bootstrap kubeconfig, which is used to mint a long-lived token for ongoing access. Only the in-cluster kubeconfig is used by controllers for this type.

API types are defined under `api/v1alpha1` and CRDs in `config/crd/bases`.

## Storage Architecture

### Shared PostgreSQL via Kine (for k8s)

For `k8s` control plane types, KubeFlex uses a single PostgreSQL instance deployed in the `kubeflex-system` namespace that is shared across multiple control planes. This approach significantly reduces resource overhead compared to running dedicated etcd instances for each control plane. The API server communicates with PostgreSQL through Kine, a translation layer that implements the etcd API while using PostgreSQL as the backing store.

PostgreSQL installation and configuration is handled via PostCreateHook Jobs rather than as a Helm subchart. This design decision provides several benefits: it avoids Helm conditional subchart issues that can occur with older Helm versions, enables OpenShift-specific templating (since values.yaml files cannot be templated in Helm subcharts), and supports runtime variable substitution and per-control-plane dynamic configuration.

For more detailed information about this architectural choice, see the [PostgreSQL Architecture Decision](../../../docs/postgresql-architecture-decision.md) document.

### Dedicated Embedded Storage (for vcluster and k3s)

For `vcluster` control plane types, the API server and embedded etcd run together in a single process, with a persistent volume mounted to ensure data durability. This embedded approach provides strong isolation for each virtual cluster while maintaining the performance benefits of co-located storage.

For `k3s` control plane types, the K3s distribution runs as a StatefulSet with its own embedded etcd and dedicated persistent volumes for data storage. The K3s server manages both the control plane components and storage in an integrated manner, providing a complete, lightweight Kubernetes cluster with strong isolation.

### Notes on OCM

The OCM-type control plane has been deprecated. For KubeStellar deployments, OCM is now preferred to run inside a `vcluster` control plane, which allows the project to track upstream OCM releases more easily and provides better isolation.

