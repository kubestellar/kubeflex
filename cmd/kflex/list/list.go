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

package list

import (
	"fmt"
	"os"
	"time"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/client"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all control planes",
		Long:  `List all control planes managed by KubeFlex`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			cp := common.NewCP(kubeconfig)
			ExecuteList(cp)
		},
	}
}

func getAge(creationTime time.Time) string {
	duration := time.Since(creationTime)
	return duration.Round(time.Second).String()
}

func ExecuteList(cp common.CP) {
	client, err := client.GetClient(cp.Kubeconfig)
	if err != nil {
		fmt.Printf("Error getting client: %s\n", err)
		os.Exit(1)
	}

	var controlPlanes tenancyv1alpha1.ControlPlaneList
	if err := client.List(cp.Ctx, &controlPlanes); err != nil {
		fmt.Printf("Error listing control planes: %s\n", err)
		os.Exit(1)
	}

	if len(controlPlanes.Items) == 0 {
		fmt.Println("No control planes found.")
		return
	}

	fmt.Println("Control Planes:")
	fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", "NAME", "SYNCED", "READY", "TYPE", "AGE")
	for _, controlPlane := range controlPlanes.Items {
		age := getAge(controlPlane.CreationTimestamp.Time)
		synced := "Unknown"
		ready := "Unknown"

		for _, condition := range controlPlane.Status.Conditions {
			if condition.Type == "Synced" {
				synced = string(condition.Status)
			}
			if condition.Type == "Ready" {
				ready = string(condition.Status)
			}
		}

		fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", controlPlane.Name, synced, ready, controlPlane.Spec.Type, age)
	}
}
