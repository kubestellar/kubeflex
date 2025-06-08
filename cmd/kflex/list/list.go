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
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			cp := common.NewCP(kubeconfig)
			return ExecuteList(cp)
		},
	}
}

func getAge(creationTime time.Time) string {
	duration := time.Since(creationTime)
	return duration.Round(time.Second).String()
}

func ExecuteList(cp common.CP) error {
	c, err := client.GetClient(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}

	var controlPlanes tenancyv1alpha1.ControlPlaneList
	if err := c.List(cp.Ctx, &controlPlanes); err != nil {
		return fmt.Errorf("error listing control planes: %s", err)
	}

	if len(controlPlanes.Items) == 0 {
		fmt.Println("No control planes found.")
		return nil
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
	return nil
}
