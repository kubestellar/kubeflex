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

package manager

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/kubestellar/kubeflex/pkg/util"
)

// ManagerCommand is the name of the manager command
const ManagerCommand = "manager"

// ClusterName is the name of the kind cluster
const ClusterName = "kubeflex"

func checkIfKoInstalled() (bool, error) {
	cmd := exec.Command("command", "-v", "ko")
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func checkIfKustomizeInstalled() (bool, error) {
	cmd := exec.Command("command", "-v", "kustomize")
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func installKo() error {
	cmd := exec.Command("go", "install", "github.com/google/ko@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install ko: %v", err)
	}
	return nil
}

func installKustomize() error {
	cmd := exec.Command("go", "install", "sigs.k8s.io/kustomize/kustomize/v5@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install kustomize: %v", err)
	}
	return nil
}

func getArchitecture() (string, error) {
	cmd := exec.Command("go", "env", "GOARCH")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture: %v", err)
	}
	return string(output), nil
}

func getGitShortCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git commit hash: %v", err)
	}
	hash := strings.Replace(string(output), "\n", "", -1)
	return hash, nil
}

func buildLocalImage(managerCommand string, imageTag string) error {
	arch, err := getArchitecture()
	if err != nil {
		return fmt.Errorf("failed to get architecture: %v", err)
	}

	platform := fmt.Sprintf("linux/%s", arch)

	cmd := exec.Command("ko", "build", "--local", "--push=false", "-B",
		fmt.Sprintf("./cmd/%s", managerCommand), "-t", imageTag, "--platform", platform)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to build local image: %v", err)
	}
	return nil

}

func loadLocalImage(managerCommand string, imageTag string, clusterName string) error {
	imageName := fmt.Sprintf("ko.local/%s:%s", managerCommand, imageTag)

	cmd := exec.Command("kind", "load", "docker-image", imageName,
		fmt.Sprintf("--name=%s", clusterName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to load local image: %v", err)
	}
	return nil
}

func install(managerCommand string, imageTag string) error {
	homeDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get the current working directory")
	}

	imageName := fmt.Sprintf("ko.local/%s:%s", managerCommand, imageTag)

	err = os.Chdir(fmt.Sprintf("%s/config/manager", homeDir))
	if err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	cmd := exec.Command("kustomize", "edit", "set", "image",
		fmt.Sprintf("controller=%s", imageName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to edit kustomization file: %v", err)
	}

	err = os.Chdir(homeDir)
	if err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	cmd = exec.Command("kubectl", "apply", "-k", "config/default")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to apply kubernetes manifests: %v", err)
	}
	return nil
}

func InstallManager() {
	done := make(chan bool)
	var wg sync.WaitGroup

	util.PrintStatus("Checking if ko is installed...", done, &wg)
	ok, err := checkIfKoInstalled()
	if err != nil {
		log.Fatalf("Error checking if ko is installed: %v\n", err)
	}
	done <- true

	if !ok {
		util.PrintStatus("Installing ko...", done, &wg)
		err = installKo()
		if err != nil {
			log.Fatalf("Error installing ko: %v\n", err)
		}
		done <- true
	}

	util.PrintStatus("Checking if kustomize is installed...", done, &wg)
	ok, err = checkIfKustomizeInstalled()
	if err != nil {
		log.Fatalf("Error checking if kustomize is installed: %v\n", err)
	}
	done <- true

	if !ok {
		util.PrintStatus("Installing kustomize...", done, &wg)
		err = installKustomize()
		if err != nil {
			log.Fatalf("Error installing kustomize: %v\n", err)
		}
		done <- true
	}

	util.PrintStatus("Building local image...", done, &wg)
	imageTag, err := getGitShortCommitHash()
	if err != nil {
		log.Fatalf("Error getting git commit hash: %v\n", err)
	}
	err = buildLocalImage(ManagerCommand, imageTag)
	if err != nil {
		log.Fatalf("Error building local image: %v\n", err)
	}
	done <- true

	util.PrintStatus("Loading local image into kind...", done, &wg)
	err = loadLocalImage(ManagerCommand, imageTag, ClusterName)
	if err != nil {
		log.Fatalf("Error loading local image into kind: %v\n", err)
	}
	done <- true

	util.PrintStatus("Installing kubeflex controller manager...", done, &wg)
	err = install(ManagerCommand, imageTag)
	if err != nil {
		log.Fatalf("Error installing application on kind: %v\n", err)
	}
	done <- true
	wg.Wait()
}
