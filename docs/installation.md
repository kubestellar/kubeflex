# KubeFlex

A powerful Kubernetes management tool that simplifies cluster operations and deployment workflows.

---

## 🚀 Quick Start

### Prerequisites

- [`kubectl`](https://kubernetes.io/docs/tasks/tools/) installed and configured
- Internet access to download binaries
- Administrative privileges to install binaries into your system $PATH

### Installation

#### 🐧 Linux / 🍎 macOS (Quick Install)
```bash
sudo bash <(curl -s https://raw.githubusercontent.com/kubestellar/kubeflex/main/scripts/install-kubeflex.sh) --ensure-folder /usr/local/bin
```

#### 🪟 Windows (with gsudo)
```bash
gsudo bash -c "curl -s https://raw.githubusercontent.com/kubestellar/kubeflex/main/scripts/install-kubeflex.sh | bash -s -- --ensure-folder /usr/local/bin"
```

📘 **Need detailed installation instructions?** See the full [Installation Guide](docs/installation.md)

## 🧰 Windows Setup (gsudo)

Windows does not come with `sudo`. You must install `gsudo`:

```bash
# Via Chocolatey
choco install gsudo

# Via Scoop
scoop install gsudo

# Via WinGet
winget install gerardog.gsudo
```

## 📖 Documentation

- [Installation Guide](docs/installation.md) – Complete setup instructions
- [Getting Started](docs/getting-started.md) – Your first steps with KubeFlex
- [API Reference](docs/api.md) – Command usage and examples

## 🤝 Contributing

We welcome contributions! Please check out our [Contributing Guide](CONTRIBUTING.md) to get started.

### 🛠 Development Setup

1. Fork this repository
2. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/kubeflex.git
   ```
3. Create a branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. Make your changes and commit:
   ```bash
   git commit -m "feat: your feature description"
   ```
5. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```
6. Open a Pull Request on GitHub

## 🆘 Support

Need help or want to discuss features?

- 📋 [Open an Issue](https://github.com/kubestellar/kubeflex/issues)
- 💬 [Start a Discussion](https://github.com/kubestellar/kubeflex/discussions)
- 📚 [Read the Docs](https://kubestellar.github.io/kubeflex)

## 📄 License

Licensed under the [Apache License 2.0](LICENSE).

## 🌟 Features

- ✅ **Cross-platform support** – Linux, macOS, and Windows
- ⚡ **One-command installation** – Quick and easy setup
- 🔗 **Kubernetes-native** – Built for seamless `kubectl` integration
- 🪶 **Lightweight** – Minimal dependencies and fast execution

## 🏗️ Project Status

KubeFlex is actively maintained by the [KubeStellar](https://github.com/kubestellar) team.

---

**Made with ❤️ by the KubeStellar community**