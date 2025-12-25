# Code Generation Guide

This document describes how to generate and maintain typed Kubernetes clients for the kubeflex CRDs.

## Overview

kubeflex uses the standard Kubernetes code-generator tools to generate:

- **Typed Clientsets**: Strongly-typed clients for interacting with CRDs via the Kubernetes API
- **Informers**: Shared informers for watching CRD resources with local caching
- **Listers**: Typed listers for reading CRD resources from informer caches

## Prerequisites

- Go 1.24+ installed
- Access to `k8s.io/code-generator` (automatically fetched during generation)

## Generating Code

To regenerate all typed clients, informers, and listers:

```bash
make generate-clients
```

This runs `hack/update-codegen.sh`, which invokes the Kubernetes code-generator tools.

## Verifying Generated Code

To verify that generated code is up-to-date:

```bash
make verify-codegen
```

This is useful in CI pipelines to ensure generated code is committed after API changes.

## Generated Structure

After running code generation, the following structure is created under `pkg/generated/`:

```
pkg/generated/
├── clientset/
│   └── versioned/
│       ├── clientset.go           # Main clientset interface
│       ├── scheme/
│       │   └── register.go        # Scheme registration
│       ├── typed/
│       │   └── v1alpha1/
│       │       ├── controlplane.go        # ControlPlane typed client
│       │       ├── postcreatehook.go      # PostCreateHook typed client
│       │       ├── v1alpha1_client.go     # Group client
│       │       ├── generated_expansion.go # Expansion interfaces
│       │       ├── doc.go
│       │       └── fake/                  # Fake clients for testing
│       └── fake/
├── informers/
│   └── externalversions/
│       ├── factory.go             # SharedInformerFactory
│       ├── generic.go             # Generic informer access
│       ├── internalinterfaces/
│       │   └── factory_interfaces.go
│       └── tenancy/
│           ├── interface.go
│           └── v1alpha1/
│               ├── interface.go
│               ├── controlplane.go    # ControlPlane informer
│               └── postcreatehook.go  # PostCreateHook informer
└── listers/
    └── tenancy/
        └── v1alpha1/
            ├── controlplane.go        # ControlPlane lister
            ├── postcreatehook.go      # PostCreateHook lister
            └── expansion_generated.go # Expansion interfaces
```

## Usage Examples

### Creating a Clientset

```go
import (
    "k8s.io/client-go/rest"
    clientset "github.com/kubestellar/kubeflex/pkg/generated/clientset/versioned"
)

func main() {
    config, err := rest.InClusterConfig()
    if err != nil {
        panic(err)
    }

    client, err := clientset.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    // Use the typed client
    controlPlanes, err := client.Tenancy().ControlPlanes("").List(ctx, metav1.ListOptions{})
}
```

### Using Informers

```go
import (
    "time"
    clientset "github.com/kubestellar/kubeflex/pkg/generated/clientset/versioned"
    informers "github.com/kubestellar/kubeflex/pkg/generated/informers/externalversions"
    "k8s.io/client-go/tools/cache"
)

func main() {
    client, _ := clientset.NewForConfig(config)
    
    // Create shared informer factory
    factory := informers.NewSharedInformerFactory(client, 30*time.Second)
    
    // Get typed informer
    cpInformer := factory.Tenancy().V1alpha1().ControlPlanes()
    
    // Add event handlers
    cpInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            // Handle add
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            // Handle update
        },
        DeleteFunc: func(obj interface{}) {
            // Handle delete
        },
    })
    
    // Start the factory
    stopCh := make(chan struct{})
    factory.Start(stopCh)
    
    // Wait for cache sync
    factory.WaitForCacheSync(stopCh)
    
    // Use the lister (reads from cache, not API server)
    lister := cpInformer.Lister()
    cp, err := lister.Get("my-control-plane")
}
```

### Using Listers Directly

```go
import (
    listers "github.com/kubestellar/kubeflex/pkg/generated/listers/tenancy/v1alpha1"
    "k8s.io/apimachinery/pkg/labels"
)

func listControlPlanes(lister listers.ControlPlaneLister) {
    // List all control planes
    cps, err := lister.List(labels.Everything())
    
    // Get a specific control plane
    cp, err := lister.Get("my-control-plane")
}
```

## When to Regenerate

Regenerate typed clients when:

1. API types in `api/v1alpha1/` are modified
2. New CRD types are added
3. Kubernetes client-go version is updated

## Troubleshooting

### Code generation fails

Ensure you have the correct version of Go installed and that `k8s.io/code-generator` is accessible.

### Generated code doesn't compile

Check that:
1. API types have proper markers (`+k8s:deepcopy-gen=package`, `+groupName=...`)
2. Types are registered with the scheme in `register.go`
3. All required `init()` functions are present

### Informers not syncing

Ensure:
1. CRDs are installed in the cluster
2. RBAC permissions allow list/watch on the resources
3. `factory.Start()` is called before `WaitForCacheSync()`
