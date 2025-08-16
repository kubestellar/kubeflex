# Copyright 2023 The KubeStellar Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# container registry
CONTAINER_REGISTRY ?= ghcr.io/kubestellar/kubeflex

# latest tag
LATEST_TAG ?= $(shell git describe --tags $(git rev-list --tags --max-count=1))

KO_DOCKER_REPO = ko.local
IMAGE_TAG ?= $(shell git rev-parse --short HEAD)
CMD_NAME ?= manager
IMG ?= ${KO_DOCKER_REPO}/${CMD_NAME}:${IMAGE_TAG}

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.26.1

# We need bash for some conditional logic below.
SHELL := /usr/bin/env bash -e

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ARCH := $(shell go env GOARCH)
OS := $(shell go env GOOS)

KUBE_CLIENT_MAJOR_VERSION := $(shell go mod edit -json | jq '.Require[] | select(.Path == "k8s.io/client-go") | .Version' --raw-output | sed 's/v\([0-9]*\).*/\1/')
KUBE_CLIENT_MINOR_VERSION := $(shell go mod edit -json | jq '.Require[] | select(.Path == "k8s.io/client-go") | .Version' --raw-output | sed "s/v[0-9]*\.\([0-9]*\).*/\1/")
GIT_COMMIT := $(shell git rev-parse --short HEAD || echo 'local')
GIT_DIRTY := $(shell git diff --quiet && echo 'clean' || echo 'dirty')
GIT_VERSION := $(shell go mod edit -json | jq '.Require[] | select(.Path == "k8s.io/client-go") | .Version' --raw-output)+kflex-$(shell git describe --tags --match='v*' --abbrev=14 "$(GIT_COMMIT)^{commit}" 2>/dev/null || echo v0.0.0-$(GIT_COMMIT))
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
MAIN_VERSION := $(shell git tag -l --sort=-v:refname | head -n1)
LDFLAGS := \
	-X github.com/kubestellar/kubeflex/cmd/kflex/common.Version=${MAIN_VERSION}.${GIT_COMMIT} \
	-X github.com/kubestellar/kubeflex/cmd/kflex/common.BuildDate=${BUILD_DATE} \
	-X k8s.io/client-go/pkg/version.gitCommit=${GIT_COMMIT} \
	-X k8s.io/client-go/pkg/version.gitTreeState=${GIT_DIRTY} \
	-X k8s.io/client-go/pkg/version.gitVersion=${GIT_VERSION} \
	-X k8s.io/client-go/pkg/version.gitMajor=${KUBE_CLIENT_MAJOR_VERSION} \
	-X k8s.io/client-go/pkg/version.gitMinor=${KUBE_CLIENT_MINOR_VERSION} \
	-X k8s.io/client-go/pkg/version.buildDate=${BUILD_DATE} \
	\
	-X k8s.io/component-base/version.gitCommit=${GIT_COMMIT} \
	-X k8s.io/component-base/version.gitTreeState=${GIT_DIRTY} \
	-X k8s.io/component-base/version.gitVersion=${GIT_VERSION} \
	-X k8s.io/component-base/version.gitMajor=${KUBE_CLIENT_MAJOR_VERSION} \
	-X k8s.io/component-base/version.gitMinor=${KUBE_CLIENT_MINOR_VERSION} \
	-X k8s.io/component-base/version.buildDate=${BUILD_DATE} \
	-extldflags '-static'
all: build
.PHONY: all

ldflags:
	@echo $(LDFLAGS)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

##@ Build

# .PHONY: build
# build: manifests generate fmt vet ## Build manager binary.
# 	go build -o bin/manager cmd/manager/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/manager/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: ko-build-local-cmupdate
ko-build-local-cmupdate: test
	ko build --local --push=false -B ./cmd/cmupdate -t ${LATEST_TAG} --platform linux/${ARCH}

.PHONY: kind-load-cmupdate-image
kind-load-cmupdate-image:
	kind load docker-image ko.local/cmupdate:${LATEST_TAG} --name kubeflex

.PHONY: ko-build-push-cmupdate
ko-build-push-cmupdate: test ## Build and push container image with ko
	KO_DOCKER_REPO=${CONTAINER_REGISTRY} ko build -B ./cmd/cmupdate -t ${LATEST_TAG},latest --platform linux/amd64,linux/arm64

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy manager to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy manager from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: chart
chart: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > chart/templates/operator.yaml
	@cat config/samples/postcreate-hooks/openshift-crds.yaml > /tmp/hooks.yaml
	@kubectl create secret generic postcreate-hooks --from-file=/tmp/hooks.yaml --dry-run=client --output=yaml > chart/templates/builtin-hooks.yaml
	@mkdir -p chart/crds
	$(KUSTOMIZE) build config/crd > chart/crds/crds.yaml

.PHONY: ko-local-build
ko-local-build:
	KO_DOCKER_REPO=${KO_DOCKER_REPO} ko build -B ./cmd/${CMD_NAME} -t ${IMAGE_TAG} --platform linux/${ARCH}

# this is used for local testing
.PHONY: kind-load-image
kind-load-image: ko-local-build
	kind load docker-image ${IMG} --name kubeflex

.PHONY: install-local-chart
install-local-chart: chart kind-load-image
	helm upgrade --install --create-namespace -n kubeflex-system kubeflex-operator ./chart

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.1.0
CONTROLLER_TOOLS_VERSION ?= v0.15.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) --output install_kustomize.sh && bash install_kustomize.sh $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); rm install_kustomize.sh; }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: verify-go-versions
verify-go-versions:
	hack/verify-go-versions.sh

.PHONY: require-%
require-%:
	@if ! command -v $* 1> /dev/null 2>&1; then echo "$* not found in ${PATH}"; exit 1; fi

.PHONY: build-all
build-all:
	GOOS=$(OS) GOARCH=$(ARCH) $(MAKE) build WHAT='./cmd/...'

.PHONY: build
build: WHAT ?= ./cmd/...
build: bin-dir require-jq require-go require-git verify-go-versions  ## Build the project
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build $(BUILDFLAGS) -ldflags="$(LDFLAGS)" -o bin $(WHAT)

.PHONY: bin-dir
bin-dir:
	mkdir -p bin
