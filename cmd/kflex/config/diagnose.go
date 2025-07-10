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

package config

import (
	"encoding/json"
	"fmt"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/spf13/cobra"
)

// DiagnosisResult represents the result of the kubeflex extension diagnosis
type DiagnosisResult struct {
	Status  string                    `json:"status"`
	Message string                    `json:"message"`
	Data    *kubeconfig.KubeflexExtensions `json:"data,omitempty"`
}

func CommandDiagnose() *cobra.Command {
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "diagnose",
		Short: "Diagnose kubeflex extension status",
		Long:  `Check the status of the global kubeflex extension in the kubeconfig file`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			flagset := cmd.Flags()
			kubeconfigFile, err := flagset.GetString(common.KubeconfigFlag)
			if err != nil {
				return fmt.Errorf("error while parsing --kubeconfig: %v", err)
			}
			return ExecuteDiagnose(kubeconfigFile, jsonOutput)
		},
	}

	command.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output in JSON format")
	return command
}

// ExecuteDiagnose checks the status of the global kubeflex extension
func ExecuteDiagnose(kubeconfigFile string, jsonOutput bool) error {
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigFile)
	if err != nil {
		return fmt.Errorf("error while loading kubeconfig: %v", err)
	}

	status, data := kubeconfig.CheckGlobalKubeflexExtension(*kconf)

	result := DiagnosisResult{
		Status:  status,
		Data:    data,
	}

	// Set appropriate message based on status
	switch status {
	case "critical":
		result.Message = "Global kubeflex extension is not present in kubeconfig"
	case "warning":
		result.Message = "Global kubeflex extension is present but empty"
	case "ok":
		result.Message = "Global kubeflex extension is present and properly configured"
	}

	if jsonOutput {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("error while marshaling JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// CLI-readable output
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Message: %s\n", result.Message)
		if data != nil && data.HostingClusterContextName != "" {
			fmt.Printf("Hosting Cluster Context: %s\n", data.HostingClusterContextName)
		}
	}

	return nil
}