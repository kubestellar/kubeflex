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

package delete

import (
	"context"
	"fmt"
	"sync"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clientK8s "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	clientKubeflex "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a control plane instance",
		Long:  `Delete a control plane instance and switches the context back to the hosting cluster context`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			cp := common.NewCP(kubeconfig, common.WithName(args[0]))
			return ExecuteDelete(cp, chattyStatus)
		},
	}
}

func ExecuteDelete(cp common.CP, chattyStatus bool) error {
	done := make(chan bool)
	controlPlane := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: cp.Name,
		},
	}
	var wg sync.WaitGroup

	util.PrintStatus(fmt.Sprintf("Deleting control plane %s...", cp.Name), done, &wg, chattyStatus)
	kconf, err := kubeconfig.LoadKubeconfig(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig: %v", err)
	}

	if err = kubeconfig.SwitchToHostingClusterContext(kconf, false); err != nil {
		return fmt.Errorf("error switching to hosting cluster kubeconfig context: %v", err)
	}

	if err := kubeconfig.WriteKubeconfig(cp.Kubeconfig, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}

	clientKflex, err := clientKubeflex.GetClient(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting kubeflex client: %v", err)
	}

	if err := clientKflex.Get(context.TODO(), clientK8s.ObjectKeyFromObject(controlPlane), controlPlane, &clientK8s.GetOptions{}); err != nil {
		return fmt.Errorf("control plane not found on server: %v", err)
	}

	if err := clientKflex.Delete(context.TODO(), controlPlane, &clientK8s.DeleteOptions{}); err != nil {
		return fmt.Errorf("error deleting instance: %v", err)
	}
	done <- true

	clientsetKflex, err := clientKubeflex.GetClientSet(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting kf client: %v", err)
	}
	util.PrintStatus(fmt.Sprintf("Waiting for control plane %s to be deleted...", cp.Name), done, &wg, chattyStatus)
	util.WaitForNamespaceDeletion(*clientsetKflex, util.GenerateNamespaceFromControlPlaneName(cp.Name))

	if controlPlane.Spec.Type != tenancyv1alpha1.ControlPlaneTypeHost &&
		controlPlane.Spec.Type != tenancyv1alpha1.ControlPlaneTypeExternal {
		if err := kubeconfig.DeleteContext(kconf, cp.Name); err != nil {
			return fmt.Errorf("no kubeconfig context for %s was found: %s", cp.Name, err)
		}

		if err := kubeconfig.WriteKubeconfig(cp.Kubeconfig, kconf); err != nil {
			return fmt.Errorf("error writing kubeconfig: %v", err)
		}
	}

	done <- true
	wg.Wait()
	return nil
}
