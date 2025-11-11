# Quick Start Guide

Get up and running with KubeFlex in 15 minutes. This guide walks you through creating your first multi-tenant control planes and experiencing the power of control-plane-as-a-service.

## What You'll Learn

By the end of this guide, you'll have:

- A running KubeFlex installation on a local Kubernetes cluster
- Multiple isolated control planes for different teams or applications
- Hands-on experience with context switching between control planes
- Understanding of how to manage control plane lifecycles

## Prerequisites

Before starting, ensure you have these tools installed:

- **kubectl** - Kubernetes command-line tool ([installation guide](https://kubernetes.io/docs/tasks/tools/))
- **kind** - Kubernetes in Docker ([installation guide](https://kind.sigs.k8s.io/))
- **KubeFlex CLI** - Follow the [installation guide](./installation.md)

**Time to complete**: Approximately 15 minutes

## Step 1: Initialize KubeFlex

Start by creating a hosting cluster with KubeFlex installed. This single command sets up everything you need:

```bash
kflex init --create-kind
```

**What happens during initialization:**

- Creates a new kind cluster named `kubeflex`
- Installs the nginx ingress controller with SSL passthrough
- Deploys the KubeFlex operator to manage control planes
- Sets up PostgreSQL as a shared storage backend
- Configures networking for local development

**Expected output:**

```
✓ Creating kind cluster kubeflex
✓ Installing ingress controller
✓ Deploying KubeFlex operator
✓ KubeFlex is ready to use!
```

> **Note**: The initialization process typically takes 2-3 minutes depending on your internet connection and system resources.

> **Important**: After running `kflex init`, do not change your kubeconfig current context using other tools (like `kubectl config use-context`) before creating your first control plane. KubeFlex needs to track the hosting cluster context.

## Step 2: Create Your First Control Plane

Now that KubeFlex is running, create your first tenant control plane:

```bash
kflex create team-alpha
```

This command creates a dedicated Kubernetes control plane named `team-alpha`. By default, KubeFlex creates a `k8s` type control plane, which is lightweight and uses shared PostgreSQL for storage.

Behind the scenes, KubeFlex:

- Creates a new namespace (`team-alpha-system`) in the hosting cluster
- Deploys an isolated API server for this control plane
- Configures a controller manager with essential Kubernetes controllers
- Generates secure access credentials
- Automatically switches your kubectl context to the new control plane

**Expected output:**

```
✓ Checking for saved hosting cluster context...
✓ Creating new control plane team-alpha...
✓ Waiting for API server to become ready...
✓ Control plane team-alpha is ready
```

> **Tip**: Control plane creation typically takes 30-60 seconds. The first one may take slightly longer as PostgreSQL initializes.

## Step 3: Interact with Your Control Plane

Your kubectl context is now set to `team-alpha`. Try some basic Kubernetes operations:

```bash
# List namespaces in team-alpha's control plane
kubectl get namespaces

# Create a new namespace
kubectl create namespace frontend

# Create another namespace
kubectl create namespace backend
```

You'll notice that your control plane starts with the standard Kubernetes namespaces (`default`, `kube-system`, etc.). The namespaces you create here are completely isolated from other control planes.

## Step 4: Create a Second Control Plane

Let's create another control plane to demonstrate isolation:

```bash
kflex create team-beta
```

KubeFlex automatically switches your context to the new `team-beta` control plane.

## Step 5: Verify Isolation

Check what namespaces exist in `team-beta`:

```bash
kubectl get namespaces
```

Notice that the `frontend` and `backend` namespaces you created in `team-alpha` are **not visible** here. Each control plane is completely isolated.

Let's create different resources in `team-beta`:

```bash
# Create a different namespace structure
kubectl create namespace api-services
kubectl create namespace database
```

## Step 6: Switch Between Control Planes

KubeFlex makes it easy to switch between control planes using the `kflex ctx` command:

```bash
# Switch back to team-alpha
kflex ctx team-alpha

# Verify you're in team-alpha by checking namespaces
kubectl get namespaces
# You'll see: default, kube-system, frontend, backend

# Switch to team-beta
kflex ctx team-beta

# Verify you're in team-beta
kubectl get namespaces
# You'll see: default, kube-system, api-services, database
```

> **Tip**: Use `kflex ctx get` to check your current context at any time.

## Step 7: Return to the Hosting Cluster

To manage KubeFlex itself or view all control planes, switch to the hosting cluster context:

```bash
# Switch back to the hosting cluster
kflex ctx
```

From the hosting cluster, you can see all your control planes:

```bash
# List all control planes
kubectl get controlplanes

# Or use the kflex CLI
kflex list
```

**Expected output:**

```
NAME         SYNCED   READY   TYPE   AGE
team-alpha   True     True    k8s    5m
team-beta    True     True    k8s    3m
```

**Understanding the output:**
- `SYNCED=True` - All resources for the control plane are successfully deployed
- `READY=True` - The API server is available and accepting requests
- `TYPE` - The control plane type (k8s, vcluster, host, or external)

## Step 8: Explore Different Control Plane Types

KubeFlex supports multiple control plane types for different use cases. Let's try a vCluster control plane:

```bash
# Create a vCluster control plane (can run actual pods)
kflex create app-workloads --type vcluster
```

**What's different about vCluster?**

Unlike the lightweight `k8s` type, vCluster control planes can schedule and run actual pod workloads using the hosting cluster's worker nodes.

Test it by creating a pod:

```bash
# Create a simple nginx pod
kubectl run nginx --image=nginx

# Wait a moment, then check if it's running
kubectl get pods
```

You'll see the nginx pod running! Switch back to the hosting cluster to see how it's actually running:

```bash
# Switch to hosting cluster
kflex ctx

# Check pods in the app-workloads namespace
kubectl get pods -n app-workloads-system
```

You'll see the nginx pod with a modified name like `nginx-x-default-x-vcluster`, showing how vCluster virtualizes resources while using the host's infrastructure.

## Step 9: Clean Up

When you're done experimenting, clean up your control planes:

```bash
# Make sure you're in the hosting context
kflex ctx

# Delete control planes
kflex delete team-alpha
kflex delete team-beta
kflex delete app-workloads
```

Each deletion removes:
- The control plane's API server
- All resources in the control plane's namespace
- Associated kubeconfig contexts
- The control plane's entry from your kubeconfig

> **Warning**: Deleting a control plane is permanent and cannot be undone. All data in that control plane will be lost.

**To clean up the entire environment:**

If you want to remove KubeFlex completely (including the kind cluster):

```bash
# Delete the kind cluster
kind delete cluster --name kubeflex
```

This removes everything, including all control planes and the hosting cluster itself.

## Understanding What You've Built

Congratulations! You've just experienced KubeFlex's core capabilities. Here's what you accomplished:

### Multi-Tenancy Without Cluster Sprawl

You created three isolated control planes (`team-alpha`, `team-beta`, `app-workloads`) on a single hosting cluster. In traditional Kubernetes, this would require three separate clusters with all the associated overhead.

### Strong Isolation

Each control plane has its own API server, ensuring that teams cannot see or interfere with each other's resources. This provides security and isolation similar to separate clusters.

### Flexible Architecture

You experienced two different control plane types:
- **k8s type**: Lightweight control planes for API-only use cases
- **vCluster type**: Full virtual clusters that can run workloads

### Simple Management

The `kflex` CLI made it easy to:
- Create control planes with a single command
- Switch between contexts seamlessly
- Clean up resources completely
