## Common Patterns and Use Cases

Now that you understand the basics, here are common patterns you can explore:

### Development Team Isolation

```bash
# Create separate control planes for different teams
kflex create frontend-team
kflex create backend-team
kflex create platform-team
```

Each team gets their own isolated environment with full admin access to their control plane.

### Environment Separation

```bash
# Create control planes for different environments
kflex create dev-environment
kflex create staging-environment
kflex create qa-environment
```

### Multi-Tenant SaaS

```bash
# Create control planes for different customers
kflex create customer-acme
kflex create customer-globex
kflex create customer-initech
```

## Troubleshooting

### Control plane creation is slow

Control plane creation typically takes 30-60 seconds. If it takes longer:

```bash
# Check the hosting cluster pods
kflex ctx
kubectl get pods -n kubeflex-system
```

Ensure the KubeFlex controller manager is running.

### Cannot switch to control plane context

If `kflex ctx <name>` fails, the hosting context reference might be missing:

```bash
# Set the current context as hosting context
kflex ctx --set-current-for-hosting
```

### Context switching doesn't work after init

If you changed your kubeconfig context between running `kflex init` and `kflex create`, you may encounter issues. This is because KubeFlex tracks the hosting cluster context. To fix:

```bash
# Use kubectl to switch to the hosting cluster
kubectl config use-context kind-kubeflex

# Set it as the hosting context
kflex ctx --set-current-for-hosting
```

### kubectl commands fail with connection errors

Ensure your control plane is ready:

```bash
kflex ctx
kubectl get controlplane <name> -o jsonpath='{.status.conditions}'
```

Both `SYNCED` and `READY` conditions should be `True`.

### Checking control plane details

To see detailed information about a control plane:

```bash
# Switch to hosting cluster
kflex ctx

# Get detailed status
kubectl describe controlplane team-alpha

# Check if API server pods are running
kubectl get pods -n team-alpha-system
```

## Next Steps

Now that you've completed the quick start, explore more advanced features:

### Learn More Concepts

- **[Architecture Guide](../concepts/architecture.md)** - Understand how KubeFlex works under the hood
- **[Overview](../concepts/overview.md)** - Deep dive into multi-tenancy and control-plane-as-a-service

### Advanced Features

- **[PostCreateHooks](../../../docs/users.md#post-create-hooks)** - Automate control plane setup with custom workflows
- **[External Clusters](../../../docs/users.md#working-with-an-external-control-plane)** - Adopt existing clusters into KubeFlex
- **[Storage Backends](../../../docs/users.md#control-plane-backends)** - Configure different storage options

### Production Deployment

- **[User Guide](../../../docs/users.md)** - Comprehensive usage documentation
- **[Installation on Production Clusters](./installation.md#installing-on-existing-clusters)** - Deploy KubeFlex on real clusters
- **[Multi-Tenancy Guide](../../../docs/multi-tenancy.md)** - Best practices for production multi-tenant deployments

## Getting Help

If you encounter issues or have questions:

- **Documentation**: Check the [User Guide](../../../docs/users.md) for detailed information
- **GitHub Issues**: [Report bugs or request features](https://github.com/kubestellar/kubeflex/issues)
- **Community Slack**: [Join the KubeStellar Slack](https://kubestellar.io/slack)
- **Examples**: Browse [sample configurations](../../../config/samples/)

## Summary

In this quick start, you:

✅ Installed KubeFlex on a local kind cluster  
✅ Created multiple isolated control planes  
✅ Experienced strong multi-tenant isolation  
✅ Switched seamlessly between control plane contexts  
✅ Explored different control plane types (k8s and vCluster)  
✅ Cleaned up resources properly

**Time invested**: ~15 minutes  
**Knowledge gained**: Production-ready multi-tenancy pattern

KubeFlex enables you to provide isolated Kubernetes environments efficiently, whether you're building a SaaS platform, managing development teams, or deploying edge infrastructure. The patterns you learned here scale from development to production deployments.

---

**Ready to dive deeper?** Head to the [User Guide](../../../docs/users.md) for comprehensive documentation and advanced use cases.
