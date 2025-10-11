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
