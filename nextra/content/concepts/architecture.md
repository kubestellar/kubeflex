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
