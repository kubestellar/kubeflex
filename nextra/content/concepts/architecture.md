## Networking & Access

KubeFlex provides multiple mechanisms for accessing tenant control planes depending on the deployment environment and access requirements.

For external access to hosted control planes, KubeFlex creates either an Ingress resource (using nginx with SSL passthrough) or an OpenShift Route, depending on the underlying platform. These resources expose the tenant API server externally, allowing users and tools to interact with the control plane from outside the hosting cluster.

For local development environments, KubeFlex defaults to using the `*.localtest.me` DNS wildcard, which conveniently resolves to 127.0.0.1 without requiring additional DNS configuration. This enables developers to quickly test multi-tenant scenarios on their local machines.

In-cluster access, used primarily by controllers and operators running within the hosting cluster, leverages the internal Kubernetes service DNS. For example, a control plane named `my-app` can be accessed at `https://my-app.my-app-system:9443` using the `cm-kubeconfig` secret. This internal access path provides better performance and doesn't traverse external ingress layers.

For cross-kind cluster networking scenarios, where multiple kind clusters need to communicate, the docker internal DNS name of the control-plane node should be used. Additional details about debugging and access patterns can be found in the [User's Guide](../../../docs/users.md) Debugging/Access sections.

## PostCreateHooks

PostCreateHooks provide a powerful mechanism for automating configuration and setup tasks after a control plane is created. Users can define a `PostCreateHook` custom resource that contains one or more Kubernetes resource templates to be applied automatically.

The control plane specification supports multiple hooks through the `spec.postCreateHooks` field, allowing administrators to compose complex initialization sequences. When rendering hook templates, KubeFlex applies a well-defined variable precedence order: system variables (such as `Namespace`, `ControlPlaneName`, and `HookName`) take the highest priority, followed by per-hook variables defined in `postCreateHooks[].vars`, then global variables from `globalVars`, and finally default variables from `PostCreateHook.spec.defaultVars`. This precedence system enables flexible configuration while maintaining sensible defaults.

Control plane creators can optionally set `waitForPostCreateHooks: true` to make the control plane's readiness status depend on the successful completion of all hooks. This ensures that the control plane is not marked as ready until all initialization tasks have finished.

Example PostCreateHook definitions can be found in [`config/samples/postcreate-hooks`](../../../config/samples/postcreate-hooks), including samples for hello-world scenarios, OpenShift CRDs, and PostgreSQL installation.

## Security Considerations

Security is a fundamental aspect of KubeFlex's multi-tenant architecture, with isolation mechanisms built in at multiple layers.

Tenant isolation is achieved by providing each control plane with its own namespace, dedicated API server, and independent RBAC configuration. This ensures that tenants cannot access or interfere with each other's resources.

Network security for external access is enforced through TLS encryption with certificate-based authentication, ensuring that only authorized users can communicate with tenant control planes. Kubeconfig secrets are automatically managed, scoped per control plane, and can be rotated to maintain security hygiene.

For external clusters managed through the `external` control plane type, long-lived tokens are generated and managed with configurable expiration policies, reducing the risk of compromised credentials. Fine-grained RBAC permissions can be configured through PostCreateHooks, allowing administrators to implement principle of least privilege tailored to each tenant's requirements.