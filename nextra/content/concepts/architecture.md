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

## Networking & Access

KubeFlex provides multiple mechanisms for accessing tenant control planes depending on the deployment environment and access requirements.

For external access to hosted control planes, KubeFlex creates either an Ingress resource (using nginx with SSL passthrough) or an OpenShift Route, depending on the underlying platform. These resources expose the tenant API server externally, allowing users and tools to interact with the control plane from outside the hosting cluster.

For local development environments, KubeFlex defaults to using the `*.localtest.me` DNS wildcard, which conveniently resolves to 127.0.0.1 without requiring additional DNS configuration. This enables developers to quickly test multi-tenant scenarios on their local machines.

In-cluster access, used primarily by controllers and operators running within the hosting cluster, leverages the internal Kubernetes service DNS. For example, a control plane named `my-app` can be accessed at `https://my-app.my-app-system:9443` using the `cm-kubeconfig` secret. This internal access path provides better performance and doesn't traverse external ingress layers.

For cross-kind cluster networking scenarios, where multiple kind clusters need to communicate, the docker internal DNS name of the control-plane node should be used. Additional details about debugging and access patterns can be found in the [User's Guide](../../../docs/users.md) Debugging/Access sections.

## PostCreateHooks

PostCreateHooks provide a powerful mechanism for automating configuration and setup tasks after a control plane is created. Users can define a `PostCreateHook` custom resource that contains one or more Kubernetes resource templates to be applied automatically.

The control plane specification supports multiple hooks through the `spec.postCreateHooks` field, allowing administrators to compose complex initialization sequences. When rendering hook templates, KubeFlex applies a well-defined variable precedence order: system variables (such as `Namespace`, `ControlPlaneName`, and `HookName`) take the highest priority, followed by per-hook variables defined in `postCreateHooks[].vars`, then global variables from `globalVars`, and finally default variables from `PostCreateHook.spec.defaultVars`. This precedence system enables flexible configuration while maintaining sensible defaults.

Control plane creators can optionally set `waitForPostCreateHooks: true` to make the control plane's readiness status depend on the successful completion of all hooks. This ensures that the control plane is not marked as ready until all initialization tasks have finished.

Example PostCreateHook definitions can be found in [`config/samples/postcreate-hooks`](../../../config/samples/postcreate-hooks), including samples for hello-world scenarios, OpenShift CRDs, and PostgreSQL installation.

## Security Considerations

Security is a fundamental aspect of KubeFlex's multi-tenant architecture, with isolation mechanisms built in at multiple layers.

Tenant isolation is achieved by providing each control plane with its own namespace, dedicated API server, and independent RBAC configuration. This ensures that tenants cannot access or interfere with each other's resources.

Network security for external access is enforced through TLS encryption with certificate-based authentication, ensuring that only authorized users can communicate with tenant control planes. Kubeconfig secrets are automatically managed, scoped per control plane, and can be rotated to maintain security hygiene.

For external clusters managed through the `external` control plane type, long-lived tokens are generated and managed with configurable expiration policies, reducing the risk of compromised credentials. Fine-grained RBAC permissions can be configured through PostCreateHooks, allowing administrators to implement principle of least privilege tailored to each tenant's requirements.

## Common Operational Notes

When working with KubeFlex, there are several operational best practices to keep in mind.

The `kflex` CLI records the hosting context name when switching between control planes, so it's important not to change the kubeconfig current context through other means (such as `kubectl config use-context`) between running `kflex init` and `kflex create`. Doing so may cause the CLI to lose track of the hosting context.

For OpenShift deployments, Route resources should be used instead of Ingress for exposing control planes. Additionally, PostgreSQL and security context configurations require conditional templating to accommodate OpenShift's stricter security policies. These platform-specific requirements are handled automatically through PostCreateHooks.

When adopting external clusters into KubeFlex management, administrators should generate a bootstrap kubeconfig with a single context. The KubeFlex controller uses this bootstrap kubeconfig to mint a long-lived token for ongoing access and then removes the bootstrap secret for security purposes.

## Next Steps

- 🚀 [Quick Start](../../../docs/quickstart.md) – Get hands-on experience
- 📖 [User's Guide](../../../docs/users.md) – Installation, CLI usage, and control plane management
- 🏗️ [Multi-Tenancy Guide](../../../docs/multi-tenancy.md) – Use cases and deployment patterns
- 📋 [PostgreSQL Architecture Decision](../../../docs/postgresql-architecture-decision.md) – Storage backend rationale

**Reference:**
- [CRDs](../../../config/crd/bases) and [API types](../../../api/v1alpha1)

*Comprehensive guides and tutorials will be added as the new documentation structure is completed.*
