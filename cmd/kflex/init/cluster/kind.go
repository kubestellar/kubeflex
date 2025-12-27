/*
Copyright 2023 The KubeStellar Authors.

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
	"bytes"
	"fmt"
	"log" // REFACTOR: why make use of log when using zap lib
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"

	"github.com/kubestellar/kubeflex/pkg/util"
)

// KindConfig is a struct that represents the kind cluster configuration
type KindConfig struct {
	Name string
}

func checkIfKindInstalled() (bool, error) {
	_, err := exec.LookPath("kind")
	if err != nil {
		return false, fmt.Errorf("failed to check kind is installed: %v", err)
	}
	return true, nil
}

func installKind() error {
	cmd := exec.Command("go", "install", "sigs.k8s.io/kind@v0.19.0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install kind: %v", err)
	}
	return nil
}

func checkKindInstanceExists(clusterName string) (bool, error) {
	cmd := exec.Command("kind", "get", "clusters")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check kind instance exists: %v", err)
	}
	if strings.Contains(string(out), clusterName) {
		return true, nil
	}
	return false, nil
}

// createKindInstance creates a kind cluster with the given name and config
func createKindInstance(name string) error {
	// create a template for the kind config file
	tmpl := template.Must(template.New("kind-config").Parse(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  ipFamily: dual
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 9080
    protocol: TCP
  - containerPort: 443
    hostPort: 9443
    protocol: TCP`))

	// create a buffer to write the template output
	var buf bytes.Buffer

	// execute the template with the name parameter
	err := tmpl.Execute(&buf, KindConfig{Name: name})
	if err != nil {
		return fmt.Errorf("failed to execute template: %v", err)
	}

	// create a temporary file to store the kind config file
	tmpFile, err := os.CreateTemp("", "kind-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// write the buffer content to the temp file
	_, err = tmpFile.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write temp file: %v", err)
	}

	// close the temp file
	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	// run the kind create cluster command with the temp file as config
	cmd := exec.Command("kind", "create", "cluster", "--name", name, "--config", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run kind create cluster command: %v", err)
	}

	return nil
}

func CreateKindCluster(chattyStatus bool, clusterName string) {
	done := make(chan bool)
	var wg sync.WaitGroup

	util.PrintStatus("Checking if kind is installed...", done, &wg, chattyStatus)
	ok, err := checkIfKindInstalled()
	if err != nil {
		log.Fatalf("Error checking if kind is installed: %v\n", err)
	}
	done <- true

	if !ok {
		util.PrintStatus("Installing kind...", done, &wg, chattyStatus)
		err = installKind()
		if err != nil {
			log.Fatalf("Error installing kind: %v\n", err)
		}
		done <- true
	}

	util.PrintStatus("Checking if a kubeflex kind instance already exists...", done, &wg, chattyStatus)
	ok, err = checkKindInstanceExists(clusterName)
	if err != nil {
		log.Fatalf("Error checking if kind instance already exists: %v\n", err)
	}
	done <- true

	if !ok {
		util.PrintStatus("Creating kind cluster...", done, &wg, chattyStatus)
		done <- true

		err = createKindInstance(clusterName)
		if err != nil {
			log.Fatalf("Error creating kind instance: %v\n", err)
		}
	}

	util.PrintStatus("Installing NGINX Gateway Fabric...", done, &wg, chattyStatus)
	err = installNGINXGatewayFabric()
	if err != nil {
		log.Fatalf("Error installing NGINX Gateway Fabric: %v\n", err)
	}
	done <- true
	wg.Wait()
}
