[![Go Report Card](https://goreportcard.com/badge/github.com/kubestellar/kubeflex)](https://goreportcard.com/report/github.com/kubestellar/kubeflex)
[![GitHub release](https://img.shields.io/github/release/kubestellar/kubeflex/all.svg?style=flat-square)](https://github.com/kubestellar/kubeflex/releases)
[![CI](https://github.com/kubestellar/kubeflex/actions/workflows/ci.yaml/badge.svg)](https://github.com/kubestellar/kubeflex/actions/workflows/ci.yaml)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=kubestellar_kubeflex&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=kubestellar_kubeflex)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=kubestellar_kubeflex&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=kubestellar_kubeflex)

# <img alt="Logo" width="90px" src="./docs/images/kubeflex-logo.png" style="vertical-align: middle;" /> KubeFlex

A flexible and scalable platform for running Kubernetes control plane APIs with multi-tenancy support.

## Overview

KubeFlex is a CNCF sandbox project under the KubeStellar umbrella that enables "control-plane-as-a-service" multi-tenancy for Kubernetes. It provides a new approach to multi-tenancy by offering each tenant their own dedicated Kubernetes control plane and data-plane nodes in a cost-effective manner.

## The Multi-Tenancy Challenge

As organizations scale their Kubernetes adoption, they face a fundamental question: how to efficiently share cluster resources across teams and applications while maintaining proper isolation, security, and cost efficiency? Traditional Kubernetes multi-tenancy approaches present significant trade-offs:

| Approach | Control Plane Isolation | Data Plane Isolation | Operational Cost | Tenant Flexibility |
|----------|------------------------|---------------------|------------------|-------------------|
| **Cluster-as-a-Service** | Full | Full | Very High | Full |
| **Namespace-as-a-Service** | None | Partial | Low | Limited |
| **Control-Plane-as-a-Service** | Full | Shared | Medium | High |
| **KubeFlex (Enhanced CaaS)** | Full | Full | Medium | High |

**The Problem**: Organizations need the isolation benefits of dedicated clusters without the operational overhead and cost. Namespace-based sharing is cost-effective but creates security and noisy-neighbor risks. Full cluster-per-tenant approaches provide excellent isolation but lead to cluster sprawl and wasted resources.

**KubeFlex's Solution**: Provides each tenant with a dedicated Kubernetes control plane (API server + controllers) while offering optional dedicated data-plane nodes through integration with KubeVirt. This approach delivers strong isolation at both control and data plane levels while maintaining cost efficiency through shared infrastructure.

*Learn more about multi-tenancy isolation approaches in this [comprehensive analysis](https://medium.com/@brauliodumba/cloud-computing-multi-tenancy-isolation-a-new-approach-815ff3e6dfd1).*

## Architecture

KubeFlex implements a sophisticated multi-tenant architecture that separates control plane management from workload execution:

![KubeFlex Architecture](./docs/images/kubeflex-architecture.png)

### Core Components

1. **KubeFlex Controller**: Orchestrates the lifecycle of tenant control planes through the ControlPlane CRD
2. **Tenant Control Planes**: Isolated API server and controller manager instances per tenant
3. **Flexible Data Plane**: Choose between shared host nodes, vCluster virtual nodes, or dedicated KubeVirt VMs
4. **Unified CLI (kflex)**: Single binary for initializing, managing, and switching between control planes
5. **Storage Abstraction**: Configurable backends from shared Postgres to dedicated etcd

### Supported Control Plane Types

- **k8s**: Lightweight Kubernetes API server (~350MB) with essential controllers, using shared Postgres via Kine
- **vcluster**: Full virtual clusters based on the vCluster project, sharing host cluster worker nodes
- **host**: The hosting cluster itself exposed as a control plane for management scenarios
- **ocm**: Open Cluster Management control plane for multi-cluster federation scenarios
- **external**: Import existing external clusters under KubeFlex management (roadmap)

### KubeFlex Scope and Third-Party Integration Boundaries

**What KubeFlex Provides:**
- Control plane provisioning and lifecycle management
- Multi-tenant API server isolation
- Flexible storage backend abstraction
- CLI tooling for tenant management
- Integration hooks for post-creation workflows

**What KubeFlex Integrates With:**
- **KubeVirt**: For VM-based worker nodes providing complete tenant isolation
- **vCluster**: As a control plane type for lightweight virtual clusters
- **Open Cluster Management**: For multi-cluster scenarios and edge deployments
- **Standard Kubernetes Storage**: CSI drivers, persistent volumes, and storage classes

**Integration Boundaries:**
KubeFlex focuses on control plane management and provides integration points rather than reimplementing existing solutions. For example, when using KubeVirt for data plane isolation, KubeFlex creates the control plane while KubeVirt handles VM provisioning and management.

## Installation

[kind](https://kind.sigs.k8s.io) and [kubectl](https://kubernetes.io/docs/tasks/tools/) are
required. A kind hosting cluster is created automatically by the kubeflex CLI. You may
also install KubeFlex on other Kube distros, as long as they support an nginx ingress
with SSL passthru, or on OpenShift. See the [User's Guide](docs/users.md) for more details.

Download the latest kubeflex CLI binary release for your OS/Architecture from the
[release page](https://github.com/kubestellar/kubeflex/releases) and copy it
to `/usr/local/bin` using the following command:

```shell
sudo su <<EOF
bash <(curl -s https://raw.githubusercontent.com/kubestellar/kubeflex/main/scripts/install-kubeflex.sh) --ensure-folder /usr/local/bin --strip-bin
EOF
```

If you have [Homebrew](https://brew.sh), use the following commands to install kubeflex:

```shell
brew tap kubestellar/kubeflex https://github.com/kubestellar/kubeflex
brew install kflex
```

To upgrade the kubeflex CLI to the latest release, you may run:

```shell
brew upgrade kflex
```

## Quick Start

### Basic Multi-Tenant Setup

Create the hosting kind cluster with ingress controller and install the kubeflex operator:

```shell
kflex init --create-kind
```

Create a control plane:

```shell
kflex create cp1
```

Interact with the new control plane, for example get namespaces and create a new one:

```shell
kflex ctx cp1
kubectl get ns
kubectl create ns myns
```

Create a second control plane and check that the namespace created in the first control plane is not present:

```shell
kflex create cp2
kflex ctx cp2
kubectl get ns
```

To go back to the hosting cluster context, use the `ctx` command:

```shell
kflex ctx
```

To switch back to a control plane context, use the `ctx <control plane name>` command, e.g:

```shell
kflex ctx cp1
```

To delete a control plane, use the `delete <control plane name>` command, e.g:

```shell
kflex delete cp1
```

### Advanced Multi-Tenant Scenario

For a realistic development team scenario with complete isolation:

1. **Initialize the hosting cluster**:
   ```shell
   kflex init --create-kind
   ```

2. **Create Team Alpha's control plane**:
   ```shell
   kflex create team-alpha --type k8s
   ```

3. **Switch to Team Alpha's isolated environment**:
   ```shell
   kflex ctx team-alpha
   kubectl create namespace frontend
   kubectl create namespace backend
   kubectl create deployment web --image=nginx -n frontend
   ```

4. **Create Team Beta's control plane**:
   ```shell
   kflex create team-beta --type k8s
   ```

5. **Switch to Team Beta's environment**:
   ```shell
   kflex ctx team-beta
   kubectl get namespaces  # Notice: team-alpha's namespaces are not visible
   kubectl create namespace api
   kubectl create deployment api-server --image=httpd -n api
   ```

6. **Verify complete isolation**:
   ```shell
   # Team Beta cannot see Team Alpha's resources
   kubectl get deployments --all-namespaces
   # Only shows Team Beta's deployments
   
   # Switch back to Team Alpha
   kflex ctx team-alpha
   kubectl get deployments --all-namespaces
   # Only shows Team Alpha's deployments
   ```

7. **Return to host cluster management**:
   ```shell
   kflex ctx
   kubectl get controlplanes
   # Shows both team-alpha and team-beta control planes
   ```

8. **Cleanup**:
   ```shell
   kflex delete team-alpha
   kflex delete team-beta
   ```

**Result**: Each team operates with complete isolation - they cannot see or interfere with each other's resources, yet they share the underlying infrastructure efficiently.

## Next Steps

Read the [User's Guide](docs/users.md) to learn more about using KubeFlex for your project
and how to create and interact with different types of control planes, such as
[vcluster](https://www.vcluster.com) and [Open Cluster Management](https://github.com/open-cluster-management-io/multicluster-controlplane).

## Use Cases and Benefits

### 1. Multi-Tenant SaaS Platforms
- **Challenge**: Provide isolated environments for hundreds of customers
- **Solution**: Create lightweight control planes per customer using the `k8s` type
- **Benefit**: Strong isolation without the cost of dedicated clusters

### 2. Enterprise Development Teams
- **Challenge**: Multiple teams need Kubernetes access without cluster sprawl
- **Solution**: Dedicated control planes with shared infrastructure
- **Benefit**: Teams get cluster-admin privileges in their own control plane

### 3. CI/CD and Testing
- **Challenge**: Isolated environments for parallel testing
- **Solution**: Ephemeral control planes created and destroyed per test run
- **Benefit**: True isolation between test runs with quick provisioning

### 4. Edge and Multi-Cluster Management
- **Challenge**: Manage multiple edge locations with varying connectivity
- **Solution**: Use `ocm` type control planes for edge cluster federation
- **Benefit**: Centralized management with distributed execution

## Advanced Configuration

### Custom Control Plane Components

```yaml
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: custom-tenant
spec:
  type: k8s
  backend: dedicated  # Use dedicated etcd instead of shared Postgres
  tokenExpirationSeconds: 7200  # 2-hour token expiration
  postCreateHooks:
    - hookName: "setup-monitoring"
      vars:
        prometheus_namespace: "monitoring"
    - hookName: "configure-networking"
      vars:
        network_policy: "strict"
```

### Storage Backend Options

1. **Shared Postgres (Default)**:
   - Multiple tenants share a Postgres instance
   - Uses Kine for etcd-compatible API
   - Most cost-effective for large numbers of tenants

2. **Dedicated etcd**:
   - Each tenant gets their own etcd instance
   - Best performance and isolation
   - Higher resource usage

3. **External Database**:
   - Connect to existing database infrastructure
   - Useful for compliance or existing investments

### Integration with KubeVirt for Data Plane Isolation

For scenarios requiring complete workload isolation:

```yaml
apiVersion: tenancy.kflex.kubestellar.org/v1alpha1
kind: ControlPlane
metadata:
  name: secure-tenant
spec:
  type: k8s
  postCreateHooks:
    - hookName: "kubevirt-nodes"
      vars:
        node_count: "3"
        vm_memory: "4Gi"
        vm_cpu: "2"
```

This creates a control plane where workloads run in dedicated KubeVirt VMs, providing:
- Complete isolation from other tenants
- Protection against container breakout attacks
- Dedicated compute resources per tenant

## Goals and Features

### Core Capabilities
- **Lightweight API Servers**: Provide dedicated Kubernetes API servers with minimal resource footprint
- **Flexible Storage Architecture**: Support shared databases, dedicated storage, or external systems
- **Custom API Server Builds**: Use upstream Kubernetes or specialized builds like multicluster-controlplane
- **Unified Management**: Single CLI for all control plane lifecycle operations

### Architecture Flexibility
- **Storage Options**: Shared Postgres, dedicated etcd, or Kine+Postgres configurations
- **API Server Variants**: Standard kubernetes API servers or trimmed-down specialized builds
- **Integration Ready**: Designed to work with existing Kubernetes ecosystem tools

### Operational Excellence
- **Zero-Touch Provisioning**: Automated control plane creation and configuration
- **Context Management**: Seamless switching between tenant environments
- **Lifecycle Management**: Complete control plane creation, update, and deletion workflows

## Documentation

- [User Guide](./docs/users.md): Detailed usage instructions and advanced scenarios
- [Architecture Guide](./docs/architecture.md): Deep-dive into technical architecture
- [Contributing Guide](./CONTRIBUTING.md): How to contribute to KubeFlex development

## Community and Support

- **Issues and Features**: [GitHub Issues](https://github.com/kubestellar/kubeflex/issues)
- **Community Discussion**: [KubeStellar Slack](https://kubestellar.slack.com)
- **Documentation**: [KubeStellar Website](https://docs.kubestellar.io/release-0.28.0/direct/kubeflex-intro/)

## License

KubeFlex is licensed under the Apache 2.0 License. See [LICENSE](./LICENSE) for the full license text.

---

*KubeFlex is part of the [KubeStellar](https://kubestellar.io) project, a CNCF sandbox initiative focused on multi-cluster configuration management for edge, multi-cloud, and hybrid cloud environments.*
