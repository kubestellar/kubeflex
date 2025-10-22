# KubeFlex Overview

KubeFlex is part of KubeStellar, which is part of the CNCF Sandbox Program. It enables "control-plane-as-a-service" multi-tenancy for Kubernetes, providing a new approach to multi-tenancy by offering each tenant their own dedicated Kubernetes control plane and data-plane nodes in a cost-effective manner.

## Why KubeFlex

KubeFlex addresses the multi-tenancy challenge by providing strong isolation of API servers and controllers for each tenant while maintaining cost efficiency. It offers a middle ground between expensive cluster-per-tenant approaches and the weaker isolation of namespace-based multi-tenancy. The platform delivers a Kubernetes-native experience through custom resource definitions and a unified command-line interface (`kflex`), supporting flexible control plane types and storage backends to meet diverse deployment requirements.

### Multi-Tenancy Problem Space

Organizations implementing Kubernetes multi-tenancy typically face a difficult trade-off. Namespace sharing provides low cost but suffers from weaker isolation, leading to noisy neighbor problems and complex RBAC management. At the other extreme, cluster-per-tenant architectures deliver strong isolation but incur high costs and significant operational overhead. KubeFlex bridges this gap by providing dedicated control planes running on shared infrastructure, achieving balanced isolation and cost efficiency.

## What KubeFlex Provides

KubeFlex delivers a complete platform for managing multi-tenant Kubernetes control planes. At its core, KubeFlex provides control plane lifecycle management through the `ControlPlane` custom resource definition, enabling declarative management of tenant control planes. A dedicated controller continuously reconciles the desired state of control planes with their actual state in the hosting cluster.

The `kflex` command-line interface offers a unified tool for initializing hosting clusters, creating and deleting control planes, switching between contexts, and managing the overall KubeFlex installation. Additionally, PostCreateHooks provide automation capabilities, allowing administrators to run templated jobs against either the hosting cluster or hosted control planes immediately after creation.

## Control Plane Types

KubeFlex supports multiple control plane types, each optimized for different use cases and operational requirements.

The **k8s** type provides a lightweight upstream Kubernetes API server with a subset of core controllers, using shared PostgreSQL via Kine for storage. This type typically consumes around 350MB of memory per instance and is ideal for scenarios where pod workloads are not needed in the control plane itself.

The **k3s** type leverages the [K3s distribution](https://k3s.io), a lightweight Kubernetes packaged as a single binary with embedded etcd and components. In KubeFlex, k3s runs as a complete, standalone Kubernetes cluster with its own control plane and data plane, using dedicated persistent volumes for data storage. It is optimized for edge deployments and resource-constrained environments while maintaining full Kubernetes compatibility.

The **vcluster** type delivers a virtual Kubernetes control plane based on the [vCluster project](https://www.vcluster.com). It runs inside a namespace of the hosting cluster and uses a syncer component to mirror workload resources to the host cluster, where they are actually executed on the host's worker nodes. The key distinction between vcluster and k3s is architectural: vcluster virtualizes only the control plane and leverages the host's data plane for workload execution, while k3s provides a complete, isolated Kubernetes cluster. This makes vcluster ideal for multi-tenancy scenarios where resource sharing is acceptable, while k3s suits use cases requiring fuller isolation. OCM functionality is supported by running OCM inside vcluster control planes rather than as a standalone type.

The **host** type exposes the hosting cluster itself under the same control plane abstraction, enabling uniform management patterns across both hosted and hosting clusters.

The **external** type allows KubeFlex to adopt an existing external Kubernetes cluster into its management framework. A bootstrap kubeconfig is used to generate a long-lived token for ongoing access, enabling centralized management of clusters created outside of KubeFlex.

For detailed information about each type, see the [User's Guide](../../../docs/users.md) and [Architecture Guide](../../../docs/architecture.md).

## How It Works

KubeFlex achieves multi-tenancy through a combination of Kubernetes-native concepts and architectural patterns that provide strong isolation while maintaining operational efficiency.

At its core, KubeFlex leverages the Kubernetes operator pattern through a custom controller that watches and reconciles `ControlPlane` custom resources. When a control plane is requested, the controller orchestrates the creation of isolated Kubernetes API servers, each running in a dedicated namespace within the hosting cluster. This namespace-based isolation provides the first layer of separation between tenants.

Each tenant control plane consists of its own API server instance and controller manager, which together form an independent Kubernetes control plane. The API server maintains its own etcd or database backend (depending on the control plane type), ensuring complete data isolation between tenants. The controller manager runs only essential Kubernetes controllers, providing the core control plane functionality without the overhead of a full cluster.

KubeFlex exposes each tenant's API server through Kubernetes Service and Ingress or Route resources, enabling external access via TLS-secured endpoints. Authentication and authorization are handled independently per tenant through dedicated kubeconfig secrets, with separate credentials for external access and in-cluster communication.

The architecture separates concerns between the control plane (managed by KubeFlex) and the data plane (where workloads execute). Depending on the control plane type, workloads can run on shared host cluster nodes (vcluster), dedicated node pools, or as completely isolated environments (k3s), providing flexibility in the isolation-versus-efficiency trade-off.
