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

const (
	OutputFormatJSON = "json"
)

// DiagnosisResult represents the result of the kubeflex extension diagnosis
type DiagnosisResult struct {
	Status  string                         `json:"status"`
	Message string                         `json:"message"`
	Data    *kubeconfig.KubeflexExtensions `json:"data,omitempty"`
}

func CommandDiagnose() *cobra.Command {
	var outputFormat string

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
			if outputFormat != OutputFormatJSON {
				return fmt.Errorf("unsupported output format: %q (only 'json' is supported)", outputFormat)
			}
			return ExecuteDiagnose(kubeconfigFile, outputFormat)
		},
	}

	command.Flags().StringVarP(&outputFormat, "output", "o", "json", "output format (json)")
	return command
}

// ExecuteDiagnose checks the status of the global kubeflex extension
func ExecuteDiagnose(kubeconfigFile string, outputFormat string) error {
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigFile)
	if err != nil {
		return fmt.Errorf("error while loading kubeconfig: %v", err)
	}

	status, data := kubeconfig.CheckGlobalKubeflexExtension(*kconf)

	result := DiagnosisResult{
		Status: status,
		Data:   data,
	}

	// Set appropriate message based on status
	switch status {
	case kubeconfig.DiagnosisStatusCritical:
		result.Message = "Global kubeflex extension is not present in kubeconfig"
	case kubeconfig.DiagnosisStatusWarning:
		result.Message = "Global kubeflex extension is present but empty"
	case kubeconfig.DiagnosisStatusOK:
		result.Message = "Global kubeflex extension is present and properly configured"
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error while marshaling JSON: %v", err)
	}
	fmt.Println(string(jsonData))
	return nil
}
