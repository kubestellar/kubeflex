# KubeFlex Installation Guide

This guide provides detailed instructions for installing KubeFlex, including prerequisites for Windows users.

## Prerequisites

Before installing KubeFlex, ensure that you have the necessary tools and permissions on your system. For Windows users, you need to install `sudo` to execute commands with elevated privileges. Here are the steps to install `sudo` on Windows:

### Installing `sudo` on Windows

1. **Using Chocolatey:**
   - If you don't have Chocolatey installed, you can install it by following the instructions on the [Chocolatey website](https://chocolatey.org/install).
   - Once Chocolatey is installed, open a command prompt as an administrator and run:
     ```bash
     choco install gsudo
     ```

2. **Using Scoop:**
   - If you don't have Scoop installed, you can install it by following the instructions on the [Scoop website](https://scoop.sh/).
   - Once Scoop is installed, open a command prompt as an administrator and run:
     ```bash
     scoop install gsudo
     ```

3. **Using WinGet:**
   - If you don't have WinGet installed, you can install it by following the instructions on the [WinGet website](https://github.com/microsoft/winget-cli).
   - Once WinGet is installed, open a command prompt as an administrator and run:
     ```bash
     winget install gerardog.gsudo
     ```

## Installation Instructions for KubeFlex

1. **Download the KubeFlex CLI:**
   - Download the latest KubeFlex CLI binary release for your OS/Architecture from the [release page](https://github.com/kubestellar/kubeflex/releases) and copy it to `/usr/local/bin` or another location in your `$PATH`.

   For example, on Linux amd64:
   ```bash
   OS_ARCH=linux_amd64
   LATEST_RELEASE_URL=\$(curl -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/kubestellar/kubeflex/releases/latest | jq -r '.assets[] | select(.name | contains("linux_amd64")) | .browser_download_url')
   sudo install -o root -g root -m 0755 <(curl -L \$LATEST_RELEASE_URL) /usr/local/bin/kfle


2.**Alternative Installation Command:**
You can also use the following command, which will automatically detect the host OS type and architecture:

```
sudo bash <(curl -s https://raw.githubusercontent.com/kubestellar/kubeflex/main/scripts/install-kubeflex.sh) --ensure-folder /usr/local/bin

```
