# Developing Kubeflex

## Prereqs

- go version >= go1.22
- git
- make 
- gcc
- docker
- kind

Make sure that `${HOME}/go/bin` is in your `$PATH`.

## How to build kubeflex from source

Clone the repo, build the binaries and add them to your path:

```shell
git clone https://github.com/kubestellar/kubeflex.git
cd kubeflex
make build-all
export PATH=$(pwd)/bin:$PATH
```

## Prepare a hosting cluster to test

One quick way to get started is just to use the kflex CLI to create
the KubeFlex hosting cluster setup with the nginx ingress and the postgres
DB:

```shell
kflex init --create-kind 
```

Once the hosting cluster is setup, you may build a local image, load it into
kind and install the local helm chart, which upgrades the existing helm chart
installed init by the `kflex init` command to use the new image (and updated CRDs
if there are CRDs changes). 

## Commands to build and load locally the kubeflex operator image for testing

Build the OCI image and store it in the local docker image registry

```shell
make ko-local-build  
```

Load the local image in kind, re-generate manifests and helm chart, and install
the helm chart:

```shell
make install-local-chart
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
