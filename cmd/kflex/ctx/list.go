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

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Printer interface for output formatting
type Printer interface {
	Header() string
	Content() string
}

// BasicTable implements Printer and fmt.Stringer for table output.
type BasicTable struct {
	Rows []BasicTableRow
}

// BasicTableRow represents a row in the context list table.
type BasicTableRow struct {
	Prefix  string
	CtxName string
	IsKflex string
	CPName  string
}

// Header returns the table header for BasicTable.
func (b BasicTable) Header() string {
	return fmt.Sprintf("%-30s %-18s %-15s\n", "CONTEXT", "MANAGED BY KFLEX", "CONTROLPLANE")
}

// Content returns the formatted table rows for BasicTable.
func (b BasicTable) Content() string {
	out := ""
	for _, row := range b.Rows {
		out += fmt.Sprintf("%s %-28s %-18s %-15s\n", row.Prefix, row.CtxName, row.IsKflex, row.CPName)
	}
	return out
}

func (b BasicTable) String() string {
	return b.Header() + b.Content()
}

func CommandList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available contexts",
		Long:  `List all available contexts in the kubeconfig file`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			kubeconfig, _ := cmd.Flags().GetString(common.KubeconfigFlag)
			cp := common.NewCP(kubeconfig)
			ExecuteCtxList(cp)
		},
	}
}

func ExecuteCtxList(cp common.CP) {
	config, err := clientcmd.LoadFromFile(cp.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	if len(config.Contexts) == 0 {
		fmt.Println("No contexts found.")
		return
	}

	currentContext := config.CurrentContext
	printer := NewBasicTablePrinter(config, currentContext)
	fmt.Print(printer.String())
}

// NewBasicTablePrinter constructs a BasicTable from kubeconfig and current context.
func NewBasicTablePrinter(config *api.Config, currentContext string) BasicTable {
	table := BasicTable{}
	for name := range config.Contexts {
		prefix := " "
		if name == currentContext {
			prefix = "*"
		}
		managed := ""
		controlPlane := ""
		kflexCtx, err := kubeconfig.NewKubeflexContextConfig(*config, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting KubeFlex extension for context '%s': %v\n", name, err)
			os.Exit(1)
		}
		if kflexCtx.Extensions != nil && kflexCtx.Extensions.ControlPlaneName != "" {
			managed = "yes"
			controlPlane = kflexCtx.Extensions.ControlPlaneName
		}
		table.Rows = append(table.Rows, BasicTableRow{
			Prefix:  prefix,
			CtxName: name,
			IsKflex: managed,
			CPName:  controlPlane,
		})
	}
	return table
}
