/*
Copyright 2025 The KubeStellar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"fmt"
	"os"
	"os/exec"
)

// installNGINXGatewayFabric installs NGINX Gateway Fabric on the kind cluster
func installNGINXGatewayFabric() error {
	// Install Gateway API CRDs
	cmd := exec.Command("kubectl", "apply", "-f", "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/standard-install.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install Gateway API CRDs: %v", err)
	}

	// Install NGINX Gateway Fabric using Helm
	cmd = exec.Command("helm", "repo", "add", "nginx-stable", "https://helm.nginx.com/stable")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to add nginx helm repo: %v", err)
	}

	cmd = exec.Command("helm", "repo", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to update helm repos: %v", err)
	}

	// Install NGINX Gateway Fabric
	cmd = exec.Command("helm", "install", "ngf", "oci://ghcr.io/nginx/charts/nginx-gateway-fabric",
		"--create-namespace", "-n", "nginx-gateway",
		"--set", "nginx.service.type=NodePort",
		"--set-json", "nginx.service.nodePorts=[{\"port\":31437,\"listenerPort\":80},{\"port\":31438,\"listenerPort\":443}]")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install NGINX Gateway Fabric: %v", err)
	}

	return nil
}
