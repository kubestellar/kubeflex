# Developing Kubeflex

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


## OCM Images

### Commands to build & publish multicluster-controlplane from fork

At this time the official multicluster-controlplane has issues and a build from
a fork is required. We expect to re-evaluate when new images will be published
by the OCM community.

1. Build and publish image from fork

```shell
git clone https://github.com/pdettori/multicluster-controlplane.git
cd multicluster-controlplane
git checkout kubeflex
make image
docker tag quay.io/open-cluster-management/multicluster-controlplane:latest quay.io/kubestellar/multicluster-controlplane:v0.2.0-kf-alpha.1
docker push quay.io/kubestellar/multicluster-controlplane:v0.2.0-kf-alpha.1
```
### Commands to package and push a chart to a OCI registry

First clone and build image as explained in [Commands to build & publish multicluster-controlplane from fork](#commands-to-build--publish-multicluster-controlplane-from-fork), then:

```shell
helm package charts/multicluster-controlplane --version v0.2.0-kf-alpha.1
helm push *.tgz oci://quay.io/kubestellar
```

### Hosting a copy of postgresql chart and image on quay.io

1. Select the desired postgresql chart tag:

   For example `CHART_TAG=13.1.5`.

2. Pull the chart from the bitmani Docker registry:

   ```shell
   helm pull oci://registry-1.docker.io/bitnamicharts/postgresql:$CHART_TAG
   ```

3. Unpack the downloaded `postgresql-$CHART_TAG.tgz` archive:

   ```shell
   tar xf postgresql-$CHART_TAG.tgz
   ```

4. In the newly created `postgresql` folder open the `values.yaml` and find the `image:` section:

   - Change the `registry:` from `docker.io` to `quay.io`.
   - Change the `repository:` from `bitnami/postgresql` to `kubestellar/postgresql`.
   - Take note of the image tag, for example `IMAGE_TAG=16.0.0-debian-11-r13`.

5. Repackage the chart in a tarball with the same name:

   ```shell
   tar czf postgresql-$CHART_TAG.tgz postgresql/
   ```

6. Push the customized chart to quay.io:

   ```shell
   helm login quay.io
   helm push postgresql-$CHART_TAG.tgz oci://quay.io/kubestellar/charts
   ```

7. Update the postgresql Helm chart reference that is hard coded in [postgresql.yaml](../chart/templates/postgresql.yaml#L97) to match the quay.io registry, repository, and tag used in the Helm push command of step 6.

8. Make a multi-arch copy of postgresql container image from `docker.io` to `quay.io` to match the customized chart image reference:

   ```shell
   docker login quay.io
   docker buildx imagetools create --tag quay.io/kubestellar/postgresql:$IMAGE_TAG docker.io/bitnami/postgresql:$IMAGE_TAG
   ```

   The registry, repository, and tag used in this command must match the values included in the customized chart at step 4.
