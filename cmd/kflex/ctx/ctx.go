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

package ctx

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/certs"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CPCtx struct {
	common.CP
	Type tenancyv1alpha1.ControlPlaneType
}

// Context switch context in Kubeconfig
func (c *CPCtx) Context(chattyStatus, failIfNone bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	kconf, err := kubeconfig.LoadKubeconfig(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	switch c.CP.Name {
	case "":
		util.PrintStatus("Checking for saved hosting cluster context...", done, &wg, chattyStatus)
		time.Sleep(1 * time.Second)
		done <- true
		if kubeconfig.IsHostingClusterContextPreferenceSet(kconf) {
			util.PrintStatus("Switching to hosting cluster context...", done, &wg, chattyStatus)
			if err = kubeconfig.SwitchToHostingClusterContext(kconf, false); err != nil {
				fmt.Fprintf(os.Stderr, "Error switching kubeconfig to hosting cluster context: %s\n", err)
				os.Exit(1)
			}
			done <- true
		} else if failIfNone {
			if !c.isCurrentContextHostingClusterContext() {
				fmt.Fprintln(os.Stderr, "The hosting cluster context is not known!\n"+
					"You can make it known to kflex by doing `kubectl config use-context $name_of_hosting_context` "+
					"and then either `kflex ctx` or `kflex create ...`")
				os.Exit(1)
			}
			util.PrintStatus("Hosting cluster context not set, setting it to current context", done, &wg, chattyStatus)
			kubeconfig.SetHostingClusterContextPreference(kconf)
			done <- true
		}
	default:
		ctxName := certs.GenerateContextName(c.Name)
		util.PrintStatus(fmt.Sprintf("Switching to context %s...", ctxName), done, &wg, chattyStatus)
		if err = kubeconfig.SwitchContext(kconf, c.Name); err != nil {
			fmt.Fprintf(os.Stderr, "kubeconfig context %s not found, trying to load from server...\n", err)
			if err := c.switchToHostingClusterContextAndWrite(kconf); err != nil {
				fmt.Fprintf(os.Stderr, "Error switching back to hosting cluster context: %s\n", err)
				os.Exit(1)
			}
			if err = c.loadAndMergeFromServer(kconf); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading kubeconfig context from server: %s\n", err)
				os.Exit(1)
			}
			// context exists only for CPs that are not of type host
			if c.Type != tenancyv1alpha1.ControlPlaneTypeHost {
				if err = kubeconfig.SwitchContext(kconf, c.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Error switching kubeconfig context after loading from server: %s\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "control plane %s is of type 'host', using hosting cluster context\n", c.Name)
				os.Exit(0)
			}
		}
		done <- true
	}

	if err = kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}
	wg.Wait()
}

func (c *CPCtx) loadAndMergeFromServer(kconfig *api.Config) error {
	kfcClientp, err := kfclient.GetClient(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kf client: %s\n", err)
		os.Exit(1)
	}
	kfcClient := *kfcClientp

	cp := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: c.CP.Name,
		},
	}
	if err := kfcClient.Get(context.TODO(), client.ObjectKeyFromObject(cp), cp, &client.GetOptions{}); err != nil {
		return fmt.Errorf("control plane not found on server: %s", err)
	}
	c.Type = cp.Spec.Type

	// for control plane of type host just switch to initial context
	if cp.Spec.Type == tenancyv1alpha1.ControlPlaneTypeHost {
		return kubeconfig.SwitchToInitialContext(kconfig, false)
	}

	// for all other control planes need to get secret with off-cluster kubeconfig
	clientsetp, err := kfclient.GetClientSet(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %s\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp

	if err := kubeconfig.LoadAndMergeNoWrite(c.Ctx, clientset, c.Name, string(cp.Spec.Type), kconfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading and merging kubeconfig: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func (c *CPCtx) switchToHostingClusterContextAndWrite(kconf *api.Config) error {
	if kubeconfig.IsHostingClusterContextPreferenceSet(kconf) {
		if err := kubeconfig.SwitchToHostingClusterContext(kconf, false); err != nil {
			return err
		}
		if err := kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
			return err
		}
	}
	return nil
}

func (c *CPCtx) isCurrentContextHostingClusterContext() bool {
	clientsetp, err := kfclient.GetClientSet(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %s\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp
	return util.CheckResourceExists(clientset, "tenancy.kflex.kubestellar.org", "v1alpha1", "controlplanes")
}
