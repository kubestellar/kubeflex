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

package shared

import "testing"

func TestBuildImageRef(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"empty version uses latest", "", baseImage + ":latest"},
		{"release build uses the semver tag", "v0.9.3.518e21f", baseImage + ":v0.9.3"},
		{"plain tag is kept", "v0.9.3", baseImage + ":v0.9.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildImageRef(tt.version); got != tt.want {
				t.Errorf("buildImageRef(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}
