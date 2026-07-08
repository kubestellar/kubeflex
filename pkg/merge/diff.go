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

// FindADiff returns one difference between the inputs, if there are any.
// The inputs are the data structure produced by json.Unmarshal
// when not given a specific output type.
// The returns are: `(found bool, left, right any, path []any, descr string)`.
func FindADiff(left, right any) (bool, any, any, []any, string) {
	switch leftTyped := left.(type) {
	case map[string]any:
		if rightMap, ok := right.(map[string]any); ok {
			for key, rightVal := range rightMap {
				if leftVal, have := leftTyped[key]; have {
					found, fLeft, fRight, fPath, descr := FindADiff(leftVal, rightVal)
					if found {
						return found, fLeft, fRight, append(fPath, key), descr
					}
				} else {
					return true, leftVal, rightVal, []any{key}, fmt.Sprintf("left lacks key %q", key)
				}
			}
			for key, leftVal := range leftTyped {
				if _, have := rightMap[key]; !have {
					return true, leftVal, nil, []any{key}, fmt.Sprintf("right lacks key %q", key)
				}
			}
			return false, nil, nil, nil, ""
		} else {
			return true, left, right, []any{}, fmt.Sprintf("left is a map, right is a %T", right)
		}
	case []any:
		if rightSlice, ok := right.([]any); ok {
			if len(leftTyped) != len(rightSlice) {
				return true, left, right, []any{}, fmt.Sprintf("left has %d members, right has %d", len(leftTyped), len(rightSlice))
			}
			for idx, rightVal := range rightSlice {
				found, fLeft, fRight, fPath, descr := FindADiff(leftTyped[idx], rightVal)
				if found {
					return true, fLeft, fRight, append(fPath, idx), descr
				}
			}
			return false, nil, nil, nil, ""
		} else {
			return true, left, right, []any{}, fmt.Sprintf("left is a slice, right is a %T", right)
		}
	case bool, float64, int64, string, nil:
		switch right.(type) {
		case bool, float64, int64, string, nil:
			return left != right, left, right, []any{}, "different atomic values"
		}
		return true, left, right, []any{}, "invalid type on right"
	}
	return true, left, right, []any{}, "invalid type on left"
}
