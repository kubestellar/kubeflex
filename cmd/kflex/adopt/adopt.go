/*
Copyright 2024 The KubeStellar Authors.

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

package adopt

import (
	"context"
	"fmt"
	"os"
	"sync"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CPAdopt struct {
	common.CP
}

// Adopt a control plane from another cluster
func (c *CPAdopt) Adopt(hook string, hookVars []string, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	controlPlaneType := tenancyv1alpha1.ControlPlaneTypeExternal

	cx.Context(chattyStatus, false, false, false)

	clp, err := kfclient.GetClient(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting client: %v\n", err)
		os.Exit(1)
	}
	cl := *clp

	cp := common.GenerateControlPlane(c.Name, string(controlPlaneType), "", hook, hookVars)

	util.PrintStatus(fmt.Sprintf("Adopting control plane %s of type %s ...", c.Name, controlPlaneType), done, &wg, chattyStatus)
	if err := cl.Create(context.TODO(), cp, &client.CreateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ControlPlane object: %v\n", err)
		os.Exit(1)
	}
	done <- true

	clientsetp, err := kfclient.GetClientSet(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp
	done <- true

	if err := kubeconfig.LoadAndMerge(c.Ctx, clientset, c.Name, string(controlPlaneType)); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading and merging kubeconfig: %s\n", err)
		os.Exit(1)
	}

	wg.Wait()
}
