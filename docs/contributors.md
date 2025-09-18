
# Developing Kubeflex

## Supported Control Plane Types

KubeFlex focuses on four main control plane types:

- **k8s**: Lightweight Kubernetes API server with essential controllers, using shared Postgres via Kine
- **vcluster**: Full virtual clusters based on the vCluster project, sharing host cluster worker nodes. 
   - *Note*: OCM (Open Cluster Management) functionality is available through vcluster integration; there is no separate OCM control plane type.
- **host**: The hosting cluster itself exposed as a control plane for management scenarios
- **external**: Import existing external clusters under KubeFlex management (roadmap)


## Prereqs

- go version >= go1.23
- git
- make
- gcc
- docker
- kind

Make sure that `${HOME}/go/bin` is in your `$PATH`.

## How to build kubeflex from source

Clone the repo, set up upstream remote, fetch tags, build the binaries and add them to your path:

```shell
# Clone your fork – command only shown for HTTPS; adjust the URL if you prefer SSH
git clone https://github.com/<your-username>/kubeflex.git
cd kubeflex

# Add the upstream repository as a remote (adjust the URL if you prefer SSH)
git remote add upstream https://github.com/kubestellar/kubeflex.git

# Fetch all tags from upstream
git fetch upstream --tags

# Build the binaries
make build-all

# Add binaries to your path
export PATH=$(pwd)/bin:$PATH
```

> **Note:** Fetching tags from upstream is important as the version information for KubeFlex binaries is derived from git tags. Without the tags, commands like `kflex init -c` (which initializes KubeFlex and creates a kind cluster) will not work correctly.

## Setting Up a Testing Cluster for KubeFlex

To prepare a hosting cluster for testing, execute the following script.
This script accomplishes several key tasks:

- Creates a new kind cluster specifically designed for the KubeFlex hosting environment.
- Configures nginx ingress with SSL passthrough capabilities to ensure secure communication.
- Builds and loads the KubeFlex Controller Manager image into the kind cluster.
- Installs a PostgreSQL database, providing the default backend for hosted API servers.
- Starts the KubeFlex controller manager.

```shell
test/e2e/setup-kubeflex.sh
```

##  Locally building cmupdate image

```shell
make ko-build-local-cmupdate
```

## Manually building and publishing cmupdate image

```shell
LATEST_TAG=<tag used for image> make ko-build-push-cmupdate
```

## Steps to make release

1. Delete branch "brew" from https://github.com/kubestellar/kubeflex, if there is such a branch.

1. `git checkout main` and make sure it (a) equals `main` in https://github.com/kubestellar/kubeflex and (b) is what you want to release.

1. check existing tags e.g.,
   ```
   git tag
   v0.1.0
   v0.1.1
   v0.2.0
   ...
   v0.3.1
   ```
1. create a new tag e.g.
   ```
   git tag v0.3.2
   ```
1. Push the tag upstream
   ```
   git push upstream --tag v0.3.2
   ```
   Wait until goreleaser completes the release process.

1. The goreleaser workflow will also create a branch named `brew` with some changes (to the homebrew instsall script) that need to get merged into `main`. Make a PR to merge `brew` into `main`, and get it approved and merged.

1. To avoid leaving a time bomb, delete that `brew` branch after it was merged into `main` (the goreleaser will fail to create the new `brew` branch if one already exists).

