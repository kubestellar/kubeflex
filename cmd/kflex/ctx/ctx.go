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
	"fmt"
	"os"
	"sync"
	"time"

	"mcc.ibm.org/kubeflex/cmd/kflex/common"
	"mcc.ibm.org/kubeflex/pkg/certs"
	"mcc.ibm.org/kubeflex/pkg/kubeconfig"
	"mcc.ibm.org/kubeflex/pkg/util"
)

type CPCtx struct {
	common.CP
}

// Context switch context in Kubeconfig
func (c *CPCtx) Context() {
	done := make(chan bool)
	var wg sync.WaitGroup
	kconf, err := kubeconfig.LoadKubeconfig(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	switch c.CP.Name {
	case "":
		util.PrintStatus("Checking for saved initial context...", done, &wg)
		time.Sleep(1 * time.Second)
		done <- true
		if kubeconfig.IsInitialConfigSet(kconf) {
			util.PrintStatus("Switching to initial context...", done, &wg)
			if err = kubeconfig.SwitchToInitialContext(kconf, false); err != nil {
				fmt.Fprintf(os.Stderr, "Error switching kubeconfig to initial context: %s\n", err)
				os.Exit(1)
			}
			done <- true
		}
	default:
		ctxName := certs.GenerateContextName(c.Name)
		util.PrintStatus(fmt.Sprintf("Switching to context %s...", ctxName), done, &wg)
		if err = kubeconfig.SwitchContext(kconf, c.Name); err != nil {
			fmt.Fprintf(os.Stderr, "Error switching kubeconfig context: %s\n", err)
			os.Exit(1)
		}
		done <- true
	}

	if err = kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}
	wg.Wait()
}
