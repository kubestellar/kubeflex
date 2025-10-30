## Installing KubeFlex on Kubernetes Clusters

Once you have the KubeFlex CLI installed, you can deploy KubeFlex on various Kubernetes distributions:

### Quick Start with Kind (Recommended for Development)

The easiest way to get started is with a new kind cluster:

```bash
# Create a kind cluster and install KubeFlex
kflex init --create-kind
```

This command will:
- Create a new kind cluster named `kubeflex`
- Install nginx ingress with SSL passthrough
- Deploy the KubeFlex operator
- Set up PostgreSQL as the shared backend

### Installing on Existing Clusters

KubeFlex can be installed on existing Kubernetes clusters that support nginx ingress with SSL passthrough.

#### Prerequisites for Existing Clusters

Your cluster must have:
- nginx ingress controller with SSL passthrough enabled
- Sufficient resources (2+ CPU cores, 4GB+ RAM)
- Storage class for persistent volumes

#### Supported Distributions

- **Kind** - Local development clusters
- **k3d** - Lightweight Kubernetes clusters
- **OpenShift** - Red Hat's Kubernetes platform
- **Standard Kubernetes** - Any CNCF-conformant distribution

#### Installation Steps

1. **Ensure nginx ingress is configured for SSL passthrough:**

```bash
# For existing nginx ingress installations
kubectl edit deployment ingress-nginx-controller -n ingress-nginx
```

Add `--enable-ssl-passthrough` to the controller arguments.

2. **Install KubeFlex:**

```bash
kflex init
```

### Installing with Helm

For advanced users or production environments, you can install KubeFlex using Helm:

```bash
# Install KubeFlex operator
helm upgrade --install kubeflex-operator \
  oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator \
  --version <latest-release-version-tag> \
  --set domain=localtest.me \
  --set externalPort=9443 \
  --namespace kubeflex-system \
  --create-namespace
```

#### Helm Installation on OpenShift

For OpenShift clusters, add the `isOpenShift=true` parameter:

```bash
helm upgrade --install kubeflex-operator \
  oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator \
  --version <latest-release-version-tag> \
  --set isOpenShift=true \
  --namespace kubeflex-system \
  --create-namespace
```

## Configuration Options

### Custom Domain Configuration

By default, KubeFlex uses `localtest.me` for local development. For production environments, configure a custom domain:

```bash
kflex init --domain your-domain.com
```

### Custom Port Configuration

For environments where port 9443 is not available:

```bash
kflex init --external-port 8443
```

## Troubleshooting

### Common Issues

#### Installation Script Fails

If the installation script fails, use an alternative method:

- macOS/Linux: install via **Homebrew** (Method 2)
- Any platform: use **Manual Download** (Method 3)

After installation, verify:
```bash
kflex version
```

#### Permission Denied Errors

If you encounter permission errors:

```bash
# Ensure the binary is executable
chmod +x /usr/local/bin/kflex

# Check file permissions
ls -la /usr/local/bin/kflex
```

#### kubectl Not Found

If kubectl is not installed:

```bash
# Install kubectl (Linux)
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install kubectl (macOS)
brew install kubectl
```

#### kind Not Found

If kind is not installed:

```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

### Getting Help

If you encounter issues not covered in this guide:

- **GitHub Issues**: [Report bugs or request features](https://github.com/kubestellar/kubeflex/issues)
- **Community Slack**: [Join the KubeStellar Slack](https://kubestellar.io/slack)
- **Documentation**: Check the [User Guide](../../../docs/users.md) for detailed usage instructions

## Next Steps

After successful installation:

1. **Follow the [Quick Start Guide](./quick-start.md)** to create your first control plane
2. **Read the [User Guide](../../../docs/users.md)** for detailed usage instructions
3. **Explore [Architecture](../concepts/architecture.md)** to understand how KubeFlex works
