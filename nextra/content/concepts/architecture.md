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
