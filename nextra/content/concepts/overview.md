# KubeFlex Overview

KubeFlex is part of KubeStellar, which is part of the CNCF Sandbox Program. It enables "control-plane-as-a-service" multi-tenancy for Kubernetes, providing a new approach to multi-tenancy by offering each tenant their own dedicated Kubernetes control plane and data-plane nodes in a cost-effective manner.

## Why KubeFlex

KubeFlex addresses the multi-tenancy challenge by providing strong isolation of API servers and controllers for each tenant while maintaining cost efficiency. It offers a middle ground between expensive cluster-per-tenant approaches and the weaker isolation of namespace-based multi-tenancy. The platform delivers a Kubernetes-native experience through custom resource definitions and a unified command-line interface (`kflex`), supporting flexible control plane types and storage backends to meet diverse deployment requirements.

### Multi-Tenancy Problem Space

Organizations implementing Kubernetes multi-tenancy typically face a difficult trade-off. Namespace sharing provides low cost but suffers from weaker isolation, leading to noisy neighbor problems and complex RBAC management. At the other extreme, cluster-per-tenant architectures deliver strong isolation but incur high costs and significant operational overhead. KubeFlex bridges this gap by providing dedicated control planes running on shared infrastructure, achieving balanced isolation and cost efficiency.

## What KubeFlex Provides

KubeFlex delivers a complete platform for managing multi-tenant Kubernetes control planes. At its core, KubeFlex provides control plane lifecycle management through the `ControlPlane` custom resource definition, enabling declarative management of tenant control planes. A dedicated controller continuously reconciles the desired state of control planes with their actual state in the hosting cluster.

The `kflex` command-line interface offers a unified tool for initializing hosting clusters, creating and deleting control planes, switching between contexts, and managing the overall KubeFlex installation. Additionally, PostCreateHooks provide automation capabilities, allowing administrators to run templated jobs against either the hosting cluster or hosted control planes immediately after creation.