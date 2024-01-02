/*
 Copyright © 2023 MicroOps-cn.

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

package w

import "sort"

type Comparable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

type SortedMapItem[K Comparable, V any] struct {
	Key   K
	Value V
}

type SortedMap[K Comparable, V any] []SortedMapItem[K, V]

func (s SortedMap[K, V]) Len() int { return len(s) }

func (s SortedMap[K, V]) Less(i, j int) bool { return s[i].Key < s[j].Key }

func (s SortedMap[K, V]) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func NewSortedMap[K Comparable, V any](kvs ...map[K]V) SortedMap[K, V] {
	m := SortedMap[K, V]{}
	for _, kvMap := range kvs {
		for k, v := range kvMap {
			m = append(m, SortedMapItem[K, V]{Key: k, Value: v})
		}
	}
	sort.Sort(m)
	return m
}

func Merge[M ~map[K]V, K comparable, V any](maps ...M) M {
	fullCap := 0
	for _, m := range maps {
		fullCap += len(m)
	}

	merged := make(M, fullCap)
	for _, m := range maps {
		for key, val := range m {
			merged[key] = val
		}
	}

	return merged
}
