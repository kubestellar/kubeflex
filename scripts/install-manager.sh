#!/bin/bash

HOME_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

MANAGER_COMMAND=manager
CLUSTER_NAME=kubeflex

check_if_ko_installed() {
  # Check if the program is installed
if ! command -v ko >/dev/null 2>&1; then
  echo "false"
  return
fi
echo "true"
}

check_if_kustomize_installed() {
  # Check if the program is installed
if ! command -v kustomize >/dev/null 2>&1; then
  echo "false"
  return
fi
echo "true"
}

install_ko() {
 go install github.com/google/ko@latest
}

install_kustomize() {
 go install sigs.k8s.io/kustomize/kustomize/v5@latest
}

get_architecture() {
  go env GOARCH
}

build_local_image() {
  arch=$(get_architecture)
  platform=linux/${arch}
  image_tag=$(git rev-parse --short HEAD)

  ko build --local --push=false -B ./cmd/${MANAGER_COMMAND} -t ${image_tag} --platform ${platform}
}

load_local_image() {
  kind load docker-image ko.local/${MANAGER_COMMAND}:$(git rev-parse --short HEAD) --name ${CLUSTER_NAME}
}

install() {
   image_tag=$(git rev-parse --short HEAD)
   IMG=ko.local/${MANAGER_COMMAND}:${image_tag}
   cd ${HOME_DIR}/config/manager
   kustomize edit set image controller=${IMG} 
   cd ${HOME_DIR}
   kubectl apply -k config/default 
}

################################################
# main ###

if [ $(check_if_ko_installed) == "false" ]; then 
   install_ko
fi  

if [ $(check_if_kustomize_installed) == "false" ]; then 
   install_kustomize
fi  

build_local_image

load_local_image

install