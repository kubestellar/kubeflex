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
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OverwriteExistingContextFlag = "overwrite-existing-context"
	SetCurrentForHostingFlag     = "set-current-for-hosting"
)

type CPCtx struct {
	common.CP
	Type    tenancyv1alpha1.ControlPlaneType
	verbose int // Doesn't need to be exported
}

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "ctx",
		Short: "Switch or get kubeconfig context",
		Long: `Running without an argument switches the context back to the hosting cluster context,
						while providing the control plane name as argument switches the context to
						that control plane. Use 'get' to retrieve the current context.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			overwriteExistingCtx, _ := flagset.GetBool(OverwriteExistingContextFlag)
			setCurrentCtxAsHosting, _ := flagset.GetBool(SetCurrentForHostingFlag)
			verbose, _ := flagset.GetInt(common.VerbosityFlag)
			cpName := ""
			if len(args) == 1 {
				cpName = args[0]
			}
			cp := CPCtx{
				CP:      common.NewCP(kubeconfig, common.WithName(cpName)),
				verbose: verbose,
			}
			return cp.ExecuteCtx(chattyStatus, true, overwriteExistingCtx, setCurrentCtxAsHosting)
		},
	}
	flagset := command.Flags()
	flagset.BoolP(OverwriteExistingContextFlag, "o", false, "Overwrite of hosting cluster context with new control plane context")
	flagset.BoolP(SetCurrentForHostingFlag, "c", false, "Set current context as hosting cluster context")
	command.AddCommand(CommandGet(), CommandList(), CommandRename(), CommandDelete())
	return command
}

// Context switch context in Kubeconfig
func (cpCtx *CPCtx) ExecuteCtx(chattyStatus, failIfNone, overwriteExistingCtx, setCurrentCtxAsHosting bool) error {
	done := make(chan bool)
	var wg sync.WaitGroup
	kconf, err := kubeconfig.LoadKubeconfig(cpCtx.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig: %v", err)

	}
	if cpCtx.Name == "" {
		// Switch to hosting cluster context when no context is provided
		if setCurrentCtxAsHosting { // set hosting cluster context unconditionally to the current context
			err = kubeconfig.SetHostingClusterContext(kconf, nil)
			if err != nil {
				return fmt.Errorf("error on ExecuteCtx: %v", err)
			}
		}
		util.PrintStatus("Checking for saved hosting cluster context...", done, &wg, chattyStatus)
		time.Sleep(1 * time.Second)
		done <- true
		if kubeconfig.IsHostingClusterContextSet(kconf) {
			util.PrintStatus("Switching to hosting cluster context...", done, &wg, chattyStatus)
			if err = kubeconfig.SwitchToHostingClusterContext(kconf); err != nil {
				return fmt.Errorf("error switching kubeconfig to hosting cluster context: %v", err)

			}
			done <- true
		} else if failIfNone {
			pclientset, err := kfclient.GetClientSet(cpCtx.Kubeconfig)
			if err != nil {
				return fmt.Errorf("error getting clientset: %v", err)
			}
			if !cpCtx.isCurrentContextHostingClusterContext(*pclientset) {
				return fmt.Errorf("the hosting cluster context is not known!\n" +
					"you can make it known to kflex by doing `kubectl config use-context` \n" +
					"to set the current context to the hosting cluster context and then using \n" +
					"`kflex ctx --set-current-for-hosting` to restore the needed kubeconfig extension")

			}
			util.PrintStatus("Hosting cluster context not set, setting it to current context", done, &wg, chattyStatus)
			err = kubeconfig.SetHostingClusterContext(kconf, nil)
			if err != nil {
				return fmt.Errorf("error on ExecuteCtx: %v", err)
			}
			done <- true
		}
	} else {
		// Switch to given context
		if overwriteExistingCtx {
			util.PrintStatus("Overwriting existing context for control plane", done, &wg, chattyStatus)
			if err = kubeconfig.DeleteAll(kconf, cpCtx.Name); err != nil {
				if cpCtx.verbose > 0 {
					fmt.Fprintf(os.Stderr, "context %s not found in kubeconfig: %s\n", cpCtx.Name, err)
				}
			}
			done <- true
		}
		util.PrintStatus(fmt.Sprintf("Switching to context %s...", cpCtx.Name), done, &wg, chattyStatus)
		if err = kubeconfig.SwitchContext(kconf, cpCtx.Name); err != nil {
			if overwriteExistingCtx && cpCtx.verbose > 0 {
				fmt.Printf("trying to load new context %s from server...\n", cpCtx.Name)
			} else {
				fmt.Fprintf(os.Stderr, "kubeconfig context %s not found (%s), trying to load from server...\n", cpCtx.Name, err)
			}
			if err := cpCtx.switchToHostingClusterContextAndWrite(kconf); err != nil {
				return fmt.Errorf("error switching back to hosting cluster context: %v", err)

			}
			if err = cpCtx.loadAndMergeFromServer(kconf); err != nil {
				return fmt.Errorf("error loading kubeconfig context from server: %v", err)

			}
			// context exists only for CPs that are not of type host
			if cpCtx.Type != tenancyv1alpha1.ControlPlaneTypeHost {
				if err = kubeconfig.SwitchContext(kconf, cpCtx.Name); err != nil {
					return fmt.Errorf("error switching kubeconfig context after loading from server: %v", err)

				}
			} else {
				fmt.Fprintf(os.Stderr, "control plane %s is of type 'host', using hosting cluster context (%s)\n", cpCtx.Name, kconf.CurrentContext)
			}
		}
		done <- true
	}
	if err = kubeconfig.WriteKubeconfig(cpCtx.Kubeconfig, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %s", err)
	}
	wg.Wait()
	return nil
}

func (cpCtx *CPCtx) loadAndMergeFromServer(kconf *api.Config) error {
	kfcClient, err := kfclient.GetClient(cpCtx.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting kf client: %s", err)

	}

	cp := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: cpCtx.CP.Name,
		},
	}
	if err := kfcClient.Get(context.TODO(), client.ObjectKeyFromObject(cp), cp, &client.GetOptions{}); err != nil {
		return fmt.Errorf("control plane not found on server: %s", err)
	}
	cpCtx.Type = cp.Spec.Type

	// for control plane of type host just switch to initial context
	if cp.Spec.Type == tenancyv1alpha1.ControlPlaneTypeHost {
		return kubeconfig.SwitchToHostingClusterContext(kconf)
	}

	// for all other control planes need to get secret with off-cluster kubeconfig
	clientsetp, err := kfclient.GetClientSet(cpCtx.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting clientset: %s", err)

	}
	clientset := *clientsetp

	if err := kubeconfig.LoadServerKubeconfigAndMergeIn(cpCtx.Ctx, kconf, clientset, cpCtx.Name, string(cp.Spec.Type)); err != nil {
		return fmt.Errorf("error loading and merging kubeconfig: %v", err)

	}
	return nil
}

func (cpCtx *CPCtx) switchToHostingClusterContextAndWrite(kconf *api.Config) error {
	if kubeconfig.IsHostingClusterContextSet(kconf) {
		if err := kubeconfig.SwitchToHostingClusterContext(kconf); err != nil {
			return err
		}
		if err := kubeconfig.WriteKubeconfig(cpCtx.Kubeconfig, kconf); err != nil {
			return err
		}
	}
	return nil
}

func (cpCtx *CPCtx) isCurrentContextHostingClusterContext(clientset kubernetes.Clientset) bool {
	return util.CheckResourceExists(clientset, "tenancy.kflex.kubestellar.org", "v1alpha1", "controlplanes")
}
