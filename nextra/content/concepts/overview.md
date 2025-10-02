# KubeFlex Overview

KubeFlex is a CNCF sandbox project under the KubeStellar umbrella that enables "control-plane-as-a-service" multi-tenancy for Kubernetes. It provides a new approach to multi-tenancy by offering each tenant their own dedicated Kubernetes control plane and data-plane nodes in a cost-effective manner.

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

Getting started with KubeFlex follows a straightforward workflow that takes users from installation to managing multiple control planes.

First, install the KubeFlex operator on a hosting cluster using `kflex init`. This command sets up the necessary components to begin hosting tenant control planes.

To create a control plane, use `kflex create <name>` or apply a `ControlPlane` custom resource with `kubectl`. The KubeFlex controller responds by creating a dedicated namespace following the `<name>-system` pattern and deploying the required components: a tenant API server, a controller manager with essential Kubernetes controllers, a Service and Ingress or Route for external access, and secrets containing both off-cluster and in-cluster kubeconfigs.

Once the control plane is ready, users can retrieve and switch to its context using `kflex ctx <name>`, enabling immediate interaction with the new tenant control plane using standard Kubernetes tools like `kubectl`.

Most control plane types share common elements including a dedicated namespace (`<cp>-system`), a Service and Ingress or Route for API server access (except for `host` and `external` types), and two kubeconfig secrets: `admin-kubeconfig` for off-cluster access and `cm-kubeconfig` for in-cluster access.

## Storage and Backends

KubeFlex provides flexible storage backends tailored to different control plane types and operational requirements.

The `k8s` control plane type uses a shared PostgreSQL instance accessed through Kine, which translates the etcd API to PostgreSQL. This shared approach dramatically reduces resource overhead while supporting hundreds of lightweight control planes.

Both `k3s` and `vcluster` control plane types use dedicated embedded storage with persistent volumes. K3s runs as a complete Kubernetes distribution with its own embedded etcd, while vcluster combines the API server and embedded etcd in a single process. These dedicated storage approaches provide stronger isolation for each control plane. Vcluster can additionally host OCM workloads.

PostgreSQL installation for the `k8s` type is implemented through PostCreateHook Jobs rather than as a Helm subchart. This design choice ensures compatibility with older Helm versions, enables OpenShift-specific templating (which isn't possible with Helm subchart values.yaml files), and provides runtime flexibility for per-control-plane variables and dynamic naming.

For a detailed explanation of this architectural decision, see the [PostgreSQL Architecture Decision](../../../docs/postgresql-architecture-decision.md) document. Additional storage configuration details can be found in the [User's Guide](../../../docs/users.md).

## PostCreateHooks

PostCreateHooks enable powerful automation by executing templated Kubernetes resources and jobs immediately after a control plane is created. Using the `PostCreateHook` custom resource definition, administrators can define resource templates that are automatically applied to either the hosting cluster (the default) or the hosted control plane (by mounting the in-cluster kubeconfig and setting the `KUBECONFIG` environment variable).

Multiple hooks can be specified in the `ControlPlane.spec.postCreateHooks` field, allowing complex initialization sequences to be composed. For scenarios where the control plane should not be considered ready until initialization is complete, setting `waitForPostCreateHooks: true` makes the control plane's readiness status depend on successful hook completion.

## Who Uses KubeFlex

KubeFlex serves a diverse range of users and use cases across the Kubernetes ecosystem.

Platform Engineers use KubeFlex to build multi-tenant platforms that serve many internal teams, providing each team with dedicated control planes while maintaining centralized management and cost efficiency.

Developers leverage KubeFlex to create isolated development and test environments that mirror production configurations without the overhead of spinning up complete Kubernetes clusters for each environment.

Operations teams rely on KubeFlex to reduce cluster sprawl while maintaining strong isolation boundaries between different applications, business units, or customers.

SaaS Providers implement KubeFlex to deliver per-customer control planes that provide excellent isolation and customization capabilities while keeping infrastructure costs manageable through shared hosting resources.

## Quick Start

Start here: [Getting Started → Quick Start](../../../docs/quickstart.md)

## Next Steps

- 📚 [Architecture](./architecture.md) – Technical details and components
- 📖 [User's Guide](../../../docs/users.md) – Installation, CLI usage, creating control planes
- 🏗️ [Multi-Tenancy Guide](../../../docs/multi-tenancy.md) – Challenges, solutions, and use cases

*More guides and tutorials coming soon as part of the new documentation structure.*

