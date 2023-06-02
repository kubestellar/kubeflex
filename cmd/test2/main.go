package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// ManagerCommand is the name of the manager command
const ManagerCommand = "manager"

// ClusterName is the name of the kind cluster
const ClusterName = "kubeflex"

// checkIfKoInstalled checks if the ko program is installed
func checkIfKoInstalled() (bool, error) {
	// run the command -v ko to check if ko exists
	cmd := exec.Command("command", "-v", "ko")
	err := cmd.Run()
	if err != nil {
		// if the command returns an error, it means ko is not installed
		return false, nil
	}
	// if the command returns no error, it means ko is installed
	return true, nil
}

// checkIfKustomizeInstalled checks if the kustomize program is installed
func checkIfKustomizeInstalled() (bool, error) {
	// run the command -v kustomize to check if kustomize exists
	cmd := exec.Command("command", "-v", "kustomize")
	err := cmd.Run()
	if err != nil {
		// if the command returns an error, it means kustomize is not installed
		return false, nil
	}
	// if the command returns no error, it means kustomize is installed
	return true, nil
}

// installKo installs the ko program using go install
func installKo() error {
	// run the go install github.com/google/ko@latest command to install ko
	cmd := exec.Command("go", "install", "github.com/google/ko@latest")
	cmd.Stdout = os.Stdout // redirect stdout to os.Stdout
	cmd.Stderr = os.Stderr // redirect stderr to os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install ko: %v", err)
	}
	return nil // no error
}

// installKustomize installs the kustomize program using go install
func installKustomize() error {
	// run the go install sigs.k8s.io/kustomize/kustomize/v5@latest command to install kustomize
	cmd := exec.Command("go", "install", "sigs.k8s.io/kustomize/kustomize/v5@latest")
	cmd.Stdout = os.Stdout // redirect stdout to os.Stdout
	cmd.Stderr = os.Stderr // redirect stderr to os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install kustomize: %v", err)
	}
	return nil // no error
}

// getArchitecture gets the architecture of the current system using go env GOARCH
func getArchitecture() (string, error) {
	// run the go env GOARCH command to get the architecture
	cmd := exec.Command("go", "env", "GOARCH")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture: %v", err)
	}
	return string(output), nil // return the output as string
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

// buildLocalImage builds a local image using ko build with the given manager command and image tag
func buildLocalImage(managerCommand string, imageTag string) error {
	// get the architecture of the current system
	arch, err := getArchitecture()
	if err != nil {
		return fmt.Errorf("failed to get architecture: %v", err)
	}

	// construct the platform argument as linux/{arch}
	platform := fmt.Sprintf("linux/%s", arch)

	// run the ko build command with the given arguments
	cmd := exec.Command("ko", "build", "--local", "--push=false", "-B",
		fmt.Sprintf("./cmd/%s", managerCommand), "-t", imageTag, "--platform", platform)
	cmd.Stdout = os.Stdout // redirect stdout to os.Stdout
	cmd.Stderr = os.Stderr // redirect stderr to os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to build local image: %v", err)
	}
	return nil // no error

}

// loadLocalImage loads a local image into kind using kind load docker-image with the given manager command and image tag
func loadLocalImage(managerCommand string, imageTag string, clusterName string) error {
	// construct the image name as ko.local/{managerCommand}:{imageTag}
	imageName := fmt.Sprintf("ko.local/%s:%s", managerCommand, imageTag)

	// run the kind load docker-image command with the given arguments
	cmd := exec.Command("kind", "load", "docker-image", imageName,
		fmt.Sprintf("--name=%s", clusterName))
	cmd.Stdout = os.Stdout // redirect stdout to os.Stdout
	cmd.Stderr = os.Stderr // redirect stderr to os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to load local image: %v", err)
	}
	return nil // no error
}

// install installs the application on kind using kubectl apply -k with the given manager command and image tag
func install(managerCommand string, imageTag string) error {
	// get the home directory of the current script using os.Getenv("HOME_DIR")
	homeDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get the current working directory")
	}

	// construct the image name as ko.local/{managerCommand}:{imageTag}
	imageName := fmt.Sprintf("ko.local/%s:%s", managerCommand, imageTag)

	// change directory to ${homeDir}/config/manager using os.Chdir()
	err = os.Chdir(fmt.Sprintf("%s/config/manager", homeDir))
	if err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	// run the kustomize edit set image controller={imageName} command to set the image name in kustomization.yaml file
	cmd := exec.Command("kustomize", "edit", "set", "image",
		fmt.Sprintf("controller=%s", imageName))
	cmd.Stdout = os.Stdout // redirect stdout to os.Stdout
	cmd.Stderr = os.Stderr // redirect stderr to os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to edit kustomization file: %v", err)
	}

	// change directory back to ${homeDir} using os.Chdir()
	err = os.Chdir(homeDir)
	if err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	// run the kubectl apply -k config/default command to apply the kubernetes manifests on kind cluster
	cmd = exec.Command("kubectl", "apply", "-k", "config/default")
	cmd.Stdout = os.Stdout // redirect stdout to os.Stdout
	cmd.Stderr = os.Stderr // redirect stderr to os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to apply kubernetes manifests: %v", err)
	}
	return nil // no error

}

func main() {
	log.Println("Checking if ko is installed...")
	ok, err := checkIfKoInstalled()
	if err != nil {
		log.Fatalf("Error checking if ko is installed: %v\n", err)
	}
	if !ok {
		log.Println("Ko is not installed. Installing it...")
		err = installKo()
		if err != nil {
			log.Fatalf("Error installing ko: %v\n", err)
		}
		log.Println("Ko installed successfully.")
	} else {
		log.Println("Ko is already installed.")
	}

	log.Println("Checking if kustomize is installed...")
	ok, err = checkIfKustomizeInstalled()
	if err != nil {
		log.Fatalf("Error checking if kustomize is installed: %v\n", err)
	}
	if !ok {
		log.Println("Kustomize is not installed. Installing it...")
		err = installKustomize()
		if err != nil {
			log.Fatalf("Error installing kustomize: %v\n", err)
		}
		log.Println("Kustomize installed successfully.")
	} else {
		log.Println("Kustomize is already installed.")
	}

	log.Println("Building local image...")
	imageTag, err := getGitShortCommitHash()
	if err != nil {
		log.Fatalf("Error getting git commit hash: %v\n", err)
	}
	err = buildLocalImage(ManagerCommand, imageTag)
	if err != nil {
		log.Fatalf("Error building local image: %v\n", err)
	}
	log.Println("Local image built successfully.")

	log.Println("Loading local image into kind...")
	err = loadLocalImage(ManagerCommand, imageTag, ClusterName)
	if err != nil {
		log.Fatalf("Error loading local image into kind: %v\n", err)
	}
	log.Println("Local image loaded into kind successfully.")

	log.Println("Installing application on kind...")
	err = install(ManagerCommand, imageTag)
	if err != nil {
		log.Fatalf("Error installing application on kind: %v\n", err)
	}
	log.Println("Application installed on kind successfully.")
}
