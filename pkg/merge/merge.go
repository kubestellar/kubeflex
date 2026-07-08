package merge

import "fmt"

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

// MergePatch does something between a JSON merge patch and
// a Kubernetes strategic merge patch.
// It is a fixed static hack that seems good enough for its usage in this module.
// It is JSON merge patch except for a slice, which is updated
// member-by-member.
// The inputs and output are the data structure produced by json.Unmarshal
// when not given a specific output type.
func MergePatch(orig, patch any) (any, error) {
	switch origTyped := orig.(type) {
	case map[string]any:
		if patchMap, ok := patch.(map[string]any); ok {
			if origTyped == nil {
				return patchMap, nil
			}
			for key, patchVal := range patchMap {
				if patchVal == nil {
					delete(origTyped, key)
				} else if origVal, have := origTyped[key]; have {
					merged, err := MergePatch(origVal, patchVal)
					if err != nil {
						return nil, fmt.Errorf("failed at key=%v: %w", key, err)
					}
					origTyped[key] = merged
				} else {
					origTyped[key] = patchVal
				}
			}
			return origTyped, nil
		} else {
			return nil, fmt.Errorf("type mismatch: orig is a map, patch is a %T", patch)
		}
	case []any:
		if patchSlice, ok := patch.([]any); ok {
			if origTyped == nil {
				return patchSlice, nil
			}
			common := min(len(origTyped), len(patchSlice))
			for idx := range common {
				if origTyped[idx] == nil {
					origTyped[idx] = patchSlice[idx]
				} else {
					merged, err := MergePatch(origTyped[idx], patchSlice[idx])
					if err != nil {
						return nil, fmt.Errorf("failed at index=%v: %w", idx, err)
					}
					origTyped[idx] = merged
				}
			}
			origTyped = append(origTyped, patchSlice[common:]...)
			return origTyped, nil
		} else {
			return nil, fmt.Errorf("type mismatch: orig is a slice, patch is a %T", patch)
		}
	case bool, float64, string, nil:
		return patch, nil
	case int64: // found in a parsed template of a PostCreateHook
		return patch, nil
	default:
		return nil, fmt.Errorf("orig is a %T, which is not valid as unmarshaled JSON", orig)
	}
}
