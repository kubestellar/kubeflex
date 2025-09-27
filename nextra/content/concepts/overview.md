# KubeFlex Overview

KubeFlex is a CNCF sandbox project under the KubeStellar umbrella that enables "control-plane-as-a-service" multi-tenancy for Kubernetes. It provides a new approach to multi-tenancy by offering each tenant their own dedicated Kubernetes control plane and data-plane nodes in a cost-effective manner.

## Why KubeFlex

- Strong isolation of API servers and controllers per tenant
- Lower cost than cluster-per-tenant; higher isolation than namespace-per-tenant
- Kubernetes-native experience via CRDs and a unified CLI (`kflex`)
- Flexible control plane types and storage backends

### Multi-Tenancy Problem Space

- Namespace sharing: low cost, weaker isolation (noisy neighbors, complex RBAC)
- Cluster per tenant: strong isolation, high cost and operational overhead
- KubeFlex: dedicated control planes on shared infra → balanced isolation and cost

## What KubeFlex Provides

- Control plane lifecycle management via the `ControlPlane` CRD
- A controller that reconciles desired control planes in the hosting cluster
- A CLI (`kflex`) to initialize, create/switch/delete control planes, and manage contexts
- PostCreateHooks to run templated jobs against the hosting or hosted control plane after creation

## Control Plane Types

- `k8s`: Lightweight upstream API server with a subset of core controllers; uses shared PostgreSQL via Kine; ~350MB memory
- `k3s`: Lightweight K3s-based API server optimized for edge and resource-constrained environments; similar to k8s but with K3s optimizations
- `vcluster`: Full virtual cluster (vCluster project) that can run pods on hosting cluster worker nodes; uses embedded etcd + PV. OCM functionality is supported by running OCM inside vcluster rather than as a standalone type.
- `host`: Expose the hosting cluster itself under the same abstraction
- `external`: Adopt an existing external cluster and manage it as a control plane (long‑lived token generated from a bootstrap kubeconfig)

See also: [User's Guide](../../../docs/users.md) (Control Plane Types) and [Architecture Guide](../../../docs/architecture.md) (Supported Control Plane Types).

## How It Works (At a Glance)

1. Install the operator on a hosting cluster: `kflex init`
2. Create a control plane: `kflex create <name>` (or apply a `ControlPlane` CR)
3. The controller creates `<name>-system` and deploys:
   - API server for the tenant
   - Controller manager with essential controllers
   - Service and Ingress/Route
   - Secrets with off‑cluster and in‑cluster kubeconfigs
4. Retrieve and switch context: `kflex ctx <name>`

Common elements across types (except `host`/`external` where noted):
- Namespace `<cp>-system`
- Service and Ingress/Route for the API server (not present for `host`/`external`)
- `admin-kubeconfig` secret (off-cluster) and `cm-kubeconfig` (in-cluster)

## Storage and Backends

Current combinations (see [User's Guide](../../../docs/users.md)):
- `k8s`: shared PostgreSQL (via Kine)
- `k3s`: shared PostgreSQL (via Kine, similar to k8s)
- `vcluster`: dedicated sqlite/embedded etcd + PV (and can host OCM)

PostgreSQL installation is implemented via PostCreateHook Jobs (not a Helm subchart) to ensure:
- Compatibility with older Helm versions and OpenShift templating needs
- Runtime flexibility (per-control-plane variables, dynamic naming)

Details: [PostgreSQL Architecture Decision](../../../docs/postgresql-architecture-decision.md).

## PostCreateHooks (Automation After Create)

Use the `PostCreateHook` CRD to define templates (Jobs/Resources) executed after a control plane is created. Hooks can target:
- Hosting cluster (default)
- Hosted control plane (by mounting the in-cluster kubeconfig and setting `KUBECONFIG`)

You can specify multiple hooks in `ControlPlane.spec.postCreateHooks` and optionally wait for completion with `waitForPostCreateHooks: true`.

## Who Uses KubeFlex

- Platform Engineers: multi-tenant platforms for many teams
- Developers: isolated development/test environments
- Operators: reduce cluster sprawl while maintaining isolation
- SaaS Providers: per-customer control planes with cost efficiency

## Quick Start

Start here: [Getting Started → Quick Start](../../../docs/quickstart.md)

## Next Steps

- 📚 [Architecture](./architecture.md) – Technical details and components
- 📖 [User's Guide](../../../docs/users.md) – Installation, CLI usage, creating control planes
- 🏗️ [Multi-Tenancy Guide](../../../docs/multi-tenancy.md) – Challenges, solutions, and use cases

*More guides and tutorials coming soon as part of the new documentation structure.*

