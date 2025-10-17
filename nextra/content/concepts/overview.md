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