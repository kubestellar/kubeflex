/*
Copyright 2026 The KubeStellar Authors.

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

package util

import "testing"

func TestParseVersionNumber(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"release build strips commit", "v0.9.3.518e21f", "v0.9.3"},
		{"double-digit minor", "v0.10.0.abc1234", "v0.10.0"},
		{"prerelease tag preserved", "v0.8.7-redux.518e21f", "v0.8.7-redux"},
		{"plain tag unchanged", "v0.9.3", "v0.9.3"},
		{"unrecognized format returned as-is", "9", "9"},
		{"empty returned as-is", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseVersionNumber(tt.in); got != tt.want {
				t.Errorf("ParseVersionNumber(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
