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

package init

const (
	URL         = "oci://registry-1.docker.io/bitnamicharts/postgresql:12.5.6"
	RepoName    = "bitnami"
	ChartName   = "postgresql"
	ReleaseName = "postgres"
)

var (
	Args = map[string]string{
		"set": "primary.extendedConfiguration=max_connections=1000,primary.priorityClassName=system-node-critical",
	}
)
