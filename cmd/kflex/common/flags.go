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

package common

import "k8s.io/client-go/tools/clientcmd"

const (
	KubeconfigFlag     = clientcmd.RecommendedConfigPathFlag // String flag
	ChattyStatusFlag   = "chatty-status"                     // Boolean flag
	VerbosityFlag      = "verbosity"                         // Int flag
	PostCreateHookFlag = "postcreate-hook"                   // String flag
	SetFlag            = "set"                               // []String flag
)

// Version injected by makefile:LDFLAGS
var Version string

// BuildDate injected by makefile:LDFLAGS
var BuildDate string

// WarningMessage injected by makefile:LDFLAGS
var WarningMessage string

type KflexGlobalOptions interface {
	WithChattyStatus(chattyStatus bool)
	WithVerbosity(verbosity int)
	WithKubeconfig(kubeconfig string)
}
