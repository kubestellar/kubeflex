# Debugging Kubeflex

## Useful Debugging Hacks

### How to open a psql command in-cluster

```shell
kubectl run -i --tty --rm debug -n kubeflex-system --image=postgres --restart=Never -- bash
psql -h mypsql-postgresql.kubeflex-system.svc -U postgres
```

### How to view certs info

```shell
openssl x509 -noout -text -in certs/apiserver.crt 
```

### Get decoded value from secret

```shell
NAMESPACE= # your namespace
NAME= # your secret name
kubectl get secrets -n ${NAMESPACE} ${NAME} -o jsonpath='{.data.apiserver\.crt}' | base64 -d
```

### How to attach a ephemeral container to debug

```shell
NAMESPACE= # your namespace
NAME= # pod name
CONTAINER= # container name
IMAGE=ubuntu:latest
kubectl debug -n ${NAMESPACE} -it ${NAME} --image=${IMAGE} --target=${CONTAINER} -- bash
# e.g. kubectl debug -n ${NAMESPACE} -it ${NAME} --image=curlimages/curl:8.1.2 --target=${CONTAINER} -- sh
```

### Getting all the command args for a process

```shell
cat /proc/<pid>/cmdline | sed -e "s/\x00/ /g"; echo
```

### Install the kubeflex operator with the OCI helm chart

```shell
helm install test -n kubeflex-system --create-namespace oci://ghcr.io/kubestellar/kubeflex/chart/kubeflex-operator --version v0.1.0
```

### How to communicate between kind clusters on the same node

One approach that is independent of local machine IP is to use the internal DNS address of
docker containers. The address is the name of the docker container. For kubflex that
address is `kubeflex-control-plane`. For example, if I have a nodeport service on 
`kubeflex-control-plane` with port 30080 and I want to access it from another kind cluster
the internal address to use is `https://kubeflex-control-plane:30080`

### Commands to build ocm from fork

```shell
git clone https://github.com/pdettori/multicluster-controlplane.git
cd multicluster-controlplane
git checkout kubeflex
make image
docker tag quay.io/open-cluster-management/multicluster-controlplane:latest quay.io/pdettori/multicluster-controlplane:latest
docker push quay.io/pdettori/multicluster-controlplane:latest
```
### Commands to package and push a chart to a OCI registry

First clone and build image as explained in [Commands to build ocm from fork](#commands-to-build-ocm-from-fork), then:

```shell
helm package charts/multicluster-controlplane
helm push multicluster-controlplane-chart-0.1.0.tgz oci://quay.io/pdettori
```

### Commands to build and load locally the kubeflex operator image for testing

```shell
ko build --local --push=false -B ./cmd/manager -t $(git rev-parse --short HEAD) --platform linux/arm64
kind load docker-image ko.local/manager:$(git rev-parse --short HEAD) --name kubeflex
```

To deploy locally the image:

```shell
make deploy IMG=ko.local/manager:$(git rev-parse --short HEAD)
```

### Commands to build and load locally the cmupdate image (for testing)

```shell
ko build --local --push=false -B ./cmd/cmupdate -t $(git rev-parse --short HEAD) --platform linux/arm64
kind load docker-image ko.local/cmupdate:$(git rev-parse --short HEAD) --name kubeflex
```
### Commands to build and push the cmupdate image

```shell
ko build --local --push=false -B ./cmd/cmupdate -t latest --platform linux/amd64,linux/arm64 
docker tag ko.local/cmupdate:latest quay.io/pdettori/cmupdate:latest
docker push quay.io/pdettori/cmupdate:latest
```

### Steps to make release

1. delete branch "brew" from https://github.com/kubestellar/kubeflex 
2. git checkout <release branch> # e.g. release-0.2
3. Run the rebase from main
```
gitr(){
  CURRENT=$(git rev-parse --abbrev-ref HEAD)
  echo "rebasing $CURRENT"
  git checkout main && git fetch upstream && git merge upstream/main && git checkout $CURRENT && git rebase main
}
```
4. git push upstream <release branch> # e.g. release-0.2
5. check existing tags e.g.,
```
git tag 
v0.1.0
v0.1.1
v0.2.0
v0.2.1
```
6. create a new tag e.g.
```
git tag v0.2.2
```
7. Push the tag upstream
```
git push upstream --tag v0.2.2
```
Finally, make a PR from the brew branch for the brew install script.