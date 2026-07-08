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

import "testing"

func TestDecideHostingCtxAction(t *testing.T) {
	cases := []struct {
		name                   string
		currentAccessesHosting bool
		storedContextSet       bool
		want                   hostingCtxAction
	}{
		// The current context wins even when a stale stored context is set.
		{"current and stored both usable -> keep current", true, true, keepCurrentContext},
		{"only current usable -> keep current", true, false, keepCurrentContext},
		{"only stored set -> switch to stored", false, true, switchToStoredContext},
		{"neither -> unknown", false, false, hostingContextUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideHostingCtxAction(tc.currentAccessesHosting, tc.storedContextSet); got != tc.want {
				t.Errorf("decideHostingCtxAction(%v, %v) = %d, want %d",
					tc.currentAccessesHosting, tc.storedContextSet, got, tc.want)
			}
		})
	}
}
