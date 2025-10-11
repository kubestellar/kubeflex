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
