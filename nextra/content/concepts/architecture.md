# KubeFlex Architecture

KubeFlex implements a sophisticated multi-tenant architecture that separates control plane management from workload execution. This page details the components, control plane lifecycle, storage backends, networking, and automation hooks.

## High-Level Diagram

![KubeFlex Architecture](../../../docs/images/kubeflex-architecture.png)

## Core Components

- KubeFlex Controller (operator):
  - Reconciles `ControlPlane` CRs
  - Creates/manages namespaces, API servers, controller managers, Services, Ingress/Routes, and kubeconfig secrets
  - Orchestrates PostCreateHooks and status reporting
- Tenant Control Planes:
  - One per tenant, isolated API server
  - Controller manager with essential controllers (namespace, gc, service accounts, etc.)
- Flexible Data Plane:
  - Shared hosting cluster nodes, vCluster virtual nodes, or dedicated KubeVirt VMs (integration point)
- CLI (`kflex`):
  - Initializes hosting cluster (optionally creates kind cluster and installs operator)
  - Creates, lists, deletes control planes; manages kubeconfig contexts (`ctx`)
- Storage Abstraction:
  - Shared PostgreSQL via Kine (for `k8s`)
  - Embedded sqlite/etcd + PV (for `vcluster`)
  - Note: OCM runs inside `vcluster` (standalone OCM type deprecated)

## Supported Control Plane Types

- `k8s`: Upstream API server + subset of controllers; no pod workloads in this control plane; ~350MB memory; shared PostgreSQL backend via Kine
- `k3s`: K3s-based lightweight API server optimized for edge deployments; similar architecture to k8s but with K3s optimizations for reduced resource footprint
- `vcluster`: Full virtual cluster capable of running pods on hosting cluster workers; API server + embedded etcd in one process; mounts a PV for persistence. OCM is run inside vcluster rather than as a standalone control plane type.
- `host`: Exposes the hosting cluster as a control plane abstraction; only in-cluster kubeconfig applies
- `external`: Represents an existing cluster not created by KubeFlex; adopted via bootstrap kubeconfig to mint a long‑lived token; only in-cluster kubeconfig is used by controllers

API types are defined under `api/v1alpha1` and CRDs in `config/crd/bases`.

## Control Plane Lifecycle (Reconcile Flow)

When a `ControlPlane` is created (via `kflex create my-app` or `kubectl apply`):

1. The operator creates a namespace `my-app-system`
2. Deploys the tenant API server pod
   - `k8s`: configured for shared PostgreSQL (in `kubeflex-system`) using Kine
   - `k3s`: similar to k8s but uses K3s optimizations; configured for shared PostgreSQL using Kine
   - `vcluster`: API server + embedded etcd as a single process; mounts a PV
   - OCM: not created as a standalone type; run OCM inside `vcluster` when needed
   - `host`/`external`: no hosted API server is created
3. Deploys a controller manager pod with essential controllers, targeting the tenant API server
4. Creates a Service and an Ingress/Route to expose the tenant API server externally
   - Not created for `host` and `external` types
5. Creates Secrets with kubeconfigs:
   - `admin-kubeconfig` (off-cluster access)
   - `cm-kubeconfig` (in-cluster access)
6. Applies PostCreateHooks (if any) and optionally waits for completion
7. Updates `.status` fields including `synced`, `ready`, and post-create hook statuses

`kflex` then fetches the off-cluster kubeconfig, merges it locally, and can switch context (`kflex ctx my-app`). For `host` control planes, it switches to the hosting cluster context.

## Storage Architecture

### Shared PostgreSQL via Kine (for `k8s`)

- A single PostgreSQL instance (in `kubeflex-system`) is shared by multiple control planes
- The API server uses Kine to speak etcd API backed by PostgreSQL
- PostgreSQL is installed/configured via PostCreateHook Jobs rather than a Helm subchart to:
  - Avoid Helm conditional subchart issues on older Helm versions
  - Enable OpenShift‑specific templating (values.yaml cannot be templated in subcharts)
  - Support runtime variable substitution and per‑control‑plane dynamic configuration

See [PostgreSQL Architecture Decision](../../../docs/postgresql-architecture-decision.md) for details.

### Notes on OCM

- The OCM-type control plane is deprecated. For KubeStellar, OCM is preferred to run inside a `vcluster` so the project can track upstream OCM releases.

### Embedded etcd/sqlite (for `vcluster`)

- API server and etcd run in one process; PV mounted for persistence

## Networking & Access

- External access for hosted control planes is provided via Ingress (nginx SSL passthrough) or OpenShift Route
- DNS defaults to `*.localtest.me` for local development (resolves to 127.0.0.1)
- In-cluster access uses the internal service `https://my-app.my-app-system:9443` and the `cm-kubeconfig`
- For cross-kind networking, use the docker internal DNS name of the control-plane node (see [User's Guide](../../../docs/users.md) Debugging/Access sections)

## PostCreateHooks

- Define a `PostCreateHook` CR with one or more templates to apply after control plane creation
- Multiple hooks per control plane are supported via `spec.postCreateHooks`
- Variable precedence when rendering templates:
  1) System vars: `Namespace`, `ControlPlaneName`, `HookName`
  2) Per-hook vars: `postCreateHooks[].vars`
  3) Global vars: `globalVars`
  4) Default vars: `PostCreateHook.spec.defaultVars`
- `waitForPostCreateHooks: true` makes readiness depend on hook completion

Examples: [`config/samples/postcreate-hooks`](../../../config/samples/postcreate-hooks) (hello, openshift‑crds, postgres)

## CLI Overview (Context Management)

- `kflex init`: install operator, optionally create kind cluster and configure ingress
- `kflex create my-app`: create control plane, wait, merge kubeconfig
- `kflex ctx [my-app]`: switch contexts; flags include `--overwrite-existing-context` and `--set-current-for-hosting`
- `kflex delete my-app`: delete control plane and clean local kubeconfig context
- `kflex list`: list control planes

Context metadata is stored in kubeconfig extensions. The hosting context name is required for switching back (`kflex ctx`). See [Hosting Context](../../../docs/users.md#hosting-context).

## Performance & Scalability

- **Resource Usage**: `k8s` type uses ~350MB memory per control plane; `k3s` optimized for lower resource usage
- **Database Scalability**: Shared PostgreSQL backend can support hundreds of lightweight control planes
- **Horizontal Scaling**: Multiple KubeFlex operators can run across different hosting clusters
- **Storage Performance**: Dedicated etcd for `vcluster` provides better performance isolation
- **Network Optimization**: In-cluster access bypasses external ingress for better latency

## Security Considerations

- **Tenant Isolation**: Each control plane runs in its own namespace with dedicated API server and RBAC
- **Network Security**: External access secured via TLS with certificate-based authentication
- **Secret Management**: Kubeconfig secrets are automatically rotated and scoped per control plane
- **Token Lifecycle**: Long-lived tokens for external clusters are managed with configurable expiration
- **RBAC**: Fine-grained permissions can be configured via PostCreateHooks

## Common Operational Notes

- Do not change kubeconfig current context between `kflex init` and `kflex create` by other means; the CLI records the hosting context when switching
- On OpenShift, use Route instead of Ingress; PostgreSQL and security contexts require conditional templating (handled by hooks)
- For external clusters, generate a bootstrap kubeconfig with a single context; the controller mints a long‑lived token and removes the bootstrap secret

## Next Steps

- 🚀 [Quick Start](../../../docs/quickstart.md) – Get hands-on experience
- 📖 [User's Guide](../../../docs/users.md) – Installation, CLI usage, and control plane management
- 🏗️ [Multi-Tenancy Guide](../../../docs/multi-tenancy.md) – Use cases and deployment patterns
- 📋 [PostgreSQL Architecture Decision](../../../docs/postgresql-architecture-decision.md) – Storage backend rationale

**Reference:**
- [CRDs](../../../config/crd/bases) and [API types](../../../api/v1alpha1)

*Comprehensive guides and tutorials will be added as the new documentation structure is completed.*

