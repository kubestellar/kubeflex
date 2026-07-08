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

// hostingCtxAction selects how ExecuteCtx resolves the hosting cluster context
// when no control plane name is given.
type hostingCtxAction int

const (
	// keepCurrentContext means the current context already accesses the hosting
	// cluster (docs/users.md precondition (a)) and wins over any stored context.
	keepCurrentContext hostingCtxAction = iota
	// switchToStoredContext falls back to the stored hosting cluster context.
	switchToStoredContext
	// hostingContextUnknown means neither is available; the caller decides what to do.
	hostingContextUnknown
)

// decideHostingCtxAction prefers the current context over the stored extension
// so create and ctx target the cluster the user is pointed at.
func decideHostingCtxAction(currentAccessesHosting, storedContextSet bool) hostingCtxAction {
	switch {
	case currentAccessesHosting:
		return keepCurrentContext
	case storedContextSet:
		return switchToStoredContext
	default:
		return hostingContextUnknown
	}
}
