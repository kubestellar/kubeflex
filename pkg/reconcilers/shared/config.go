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

package shared

const (
	// DefaultPort is the standard HTTPS port used for secure communication.
	DefaultPort = 443

	// SecurePort is the custom secure port used by KubeStellar components
	// for internal communication.
	SecurePort = 9444

	// CMHealthzPort is the port used by the controller manager to expose
	// its /healthz endpoint for readiness and liveness probes.
	CMHealthzPort = 10257
)
