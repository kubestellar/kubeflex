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
	"os"
	"sync"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a control plane instance",
		Long: `Delete a control plane instance and switches the context back to
				the hosting cluster context`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			cp := common.NewCP(kubeconfig, common.WithName(args[0]))
			ExecuteDelete(cp, chattyStatus)
		},
	}
}

func ExecuteDelete(cp common.CP, chattyStatus bool) {
	done := make(chan bool)
	controlPlane := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: cp.Name,
		},
	}
	var wg sync.WaitGroup

	util.PrintStatus(fmt.Sprintf("Deleting control plane %s...", cp.Name), done, &wg, chattyStatus)
	kconf, err := kubeconfig.LoadKubeconfig(cp.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	if err = kubeconfig.SwitchToHostingClusterContext(kconf, true); err != nil {
		fmt.Fprintf(os.Stderr, "error switching to hosting cluster kubeconfig context: %s\n", err)
		os.Exit(1)
	}

	if err := kubeconfig.WriteKubeconfig(cp.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}

	kfcClient, err := kfclient.GetClient(cp.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kf client: %s\n", err)
		os.Exit(1)
	}

	if err := kfcClient.Get(context.TODO(), client.ObjectKeyFromObject(controlPlane), controlPlane, &client.GetOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Control plane %s not found: %s\n", cp.Name, err)
	} else {
		if err := kfcClient.Delete(context.TODO(), controlPlane, &client.DeleteOptions{}); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting instance: %s\n", err)
			os.Exit(1)
		}
	}
	done <- true

	clientsetp, err := kfclient.GetClientSet(cp.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kf client: %s\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp
	util.PrintStatus(fmt.Sprintf("Waiting for control plane %s to be deleted...", cp.Name), done, &wg, chattyStatus)
	util.WaitForNamespaceDeletion(clientset, util.GenerateNamespaceFromControlPlaneName(cp.Name))

	if err := kubeconfig.DeleteContext(kconf, cp.Name); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing kubeconfig context: %s\n", err)
	}

	if err := kubeconfig.WriteKubeconfig(cp.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}

	done <- true
	wg.Wait()
}
