# Kubeflex Installation

This guide covers installing KubeFlex on your system. KubeFlex provides multiple installation methods to suit different environments and preferences.

## Prerequisites

Before installing KubeFlex, ensure you have the following tools installed:

### Required Tools

- **[kubectl](https://kubernetes.io/docs/tasks/tools/)** - Kubernetes command-line tool
- **[kind](https://kind.sigs.k8s.io/)** - Kubernetes in Docker for local development
- **[Docker](https://docs.docker.com/get-docker/)** - Container runtime (required by kind)

### Optional Tools

- **[Helm](https://helm.sh/)** - Package manager for Kubernetes (for advanced installations)
- **[Homebrew](https://brew.sh/)** - Package manager for macOS/Linux (for easy CLI installation)

### System Requirements

- **CPU**: 2+ cores recommended
- **Memory**: 4GB+ RAM recommended
- **Storage**: 10GB+ free disk space
- **Operating System**: Linux, macOS, or Windows (with WSL2)

## Installation Methods

Choose the installation method that best fits your environment:

### Method 1: Installation Script

KubeFlex provides an installation script that downloads and installs the CLI binary. This method is convenient for most users.

```bash
sudo su <<EOF
bash <(curl -s https://raw.githubusercontent.com/kubestellar/kubeflex/main/scripts/install-kubeflex.sh) --ensure-folder /usr/local/bin --strip-bin
EOF
```

### Method 2: Homebrew (macOS/Linux)

If you're using macOS or Linux with Homebrew, you can install KubeFlex using the package manager:

```bash
# Add the KubeFlex tap
brew tap kubestellar/kubeflex https://github.com/kubestellar/kubeflex

# Install KubeFlex
brew install kflex
```

To upgrade to the latest version:

```bash
brew upgrade kflex
```

### Method 3: Manual Download

Download the latest KubeFlex CLI for your OS/architecture from the [releases page](https://github.com/kubestellar/kubeflex/releases) and place it on your PATH. After downloading and extracting:

```bash
# Linux/macOS (if the archive contains a bin/kflex file)
sudo install -o root -g root -m 0755 bin/kflex /usr/local/bin/kflex

# Or, if you downloaded a single kflex binary
sudo mv kflex /usr/local/bin/kflex
sudo chmod 0755 /usr/local/bin/kflex

# Verify
kflex version
```

## Verify Installation

After installation, verify that KubeFlex is working correctly:

```bash
# Check version
kflex version

# Check help
kflex --help
```
