//go:build go1.18

/*
Copyright © 2022 MicroOps-cn.

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

package sets

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
)

type Comparable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

func New[T Comparable](items ...T) Set[T] {
	s := Set[T]{}
	s.Insert(items...)
	return s
}

type Set[T Comparable] map[T]int

func (s Set[T]) String() string {
	return fmt.Sprintf("%v", s.List())
}

// Insert adds items to the set.
func (s Set[T]) Insert(items ...T) Set[T] {
	for _, item := range items {
		if _, ok := s[item]; !ok {
			s[item] = s.Len()
		}
	}
	return s
}

// Delete removes all items from the set.
func (s Set[T]) Delete(items ...T) Set[T] {
	for _, item := range items {
		delete(s, item)
	}
	return s
}

// Has returns true if and only if item is contained in the set.
func (s Set[T]) Has(item T) bool {
	_, contained := s[item]
	return contained
}

// Clone Clone a set
func (s Set[T]) Clone() Set[T] {
	return New[T](s.List()...)
}

// HasAll returns true if and only if all items are contained in the set.
func (s Set[T]) HasAll(items ...T) bool {
	for _, item := range items {
		if !s.Has(item) {
			return false
		}
	}
	return true
}

// HasAny returns true if any items are contained in the set.
func (s Set[T]) HasAny(items ...T) bool {
	for _, item := range items {
		if s.Has(item) {
			return true
		}
	}
	return false
}

// Difference returns a set of objects that are not in s2
// For example:
// s1 = {a1, a2, a3}
// s2 = {a1, a2, a4, a5}
// s1.Difference(s2) = {a3}
// s2.Difference(s1) = {a4, a5}
func (s Set[T]) Difference(s2 Set[T]) Set[T] {
	result := New[T]()
	for key := range s {
		if !s2.Has(key) {
			result.Insert(key)
		}
	}
	return result
}

// Union returns a new set which includes items in either s1 or s2.
// For example:
// s1 = {a1, a2}
// s2 = {a3, a4}
// s1.Union(s2) = {a1, a2, a3, a4}
// s2.Union(s1) = {a1, a2, a3, a4}
func (s Set[T]) Union(s2 Set[T]) Set[T] {
	result := New[T]()
	for key := range s {
		result.Insert(key)
	}
	for key := range s2 {
		result.Insert(key)
	}
	return result
}

// Intersection returns a new set which includes the item in BOTH s1 and s2
// For example:
// s1 = {a1, a2}
// s2 = {a2, a3}
// s1.Intersection(s2) = {a2}
func (s Set[T]) Intersection(s2 Set[T]) Set[T] {
	var walk, other Set[T]
	result := New[T]()
	if s.Len() < s2.Len() {
		walk = s
		other = s2
	} else {
		walk = s2
		other = s
	}
	for key := range walk {
		if other.Has(key) {
			result.Insert(key)
		}
	}
	return result
}

// IsSuperset returns true if and only if s1 is a superset of s2.
func (s Set[T]) IsSuperset(s2 Set[T]) bool {
	for item := range s2 {
		if !s.Has(item) {
			return false
		}
	}
	return true
}

type sortableSlice[T Comparable] []T

func (s sortableSlice[T]) Len() int           { return len(s) }
func (s sortableSlice[T]) Less(i, j int) bool { return less[T](s[i], s[j]) }
func (s sortableSlice[T]) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Equal returns true if and only if s1 is equal (as a set) to s2.
// Two sets are equal if their membership is identical.
// (In practice, this means same elements, order doesn't matter)
func (s Set[T]) Equal(s2 Set[T]) bool {
	return s.Len() == s2.Len() && s.IsSuperset(s2)
}

// SortedList returns the slice with contents in random order.
func (s Set[T]) SortedList() []T {
	res := make([]T, 0, s.Len())
	for key := range s {
		res = append(res, key)
	}
	sort.Sort(sortableSlice[T](res))
	return res
}

// List returns the contents as a sorted string slice.
func (s Set[T]) List() []T {
	res := make(sortableSet[T], 0, s.Len())

	for item, index := range s {
		res = append([]setItem[T](res), setItem[T]{index: index, item: item})
	}
	sort.Sort(res)
	return res.List()
}

// PopAny returns a single element from the set.
func (s Set[T]) PopAny() (T, bool) {
	for key := range s {
		s.Delete(key)
		return key, true
	}
	var zeroValue T
	return zeroValue, false
}

// Len returns the size of the set.
func (s Set[T]) Len() int {
	return len(s)
}

func less[T Comparable](lhs, rhs T) bool {
	return lhs < rhs
}

type setItem[T Comparable] struct {
	index int
	item  T
}

type sortableSet[T Comparable] []setItem[T]

func (s sortableSet[T]) Len() int {
	return len(s)
}

func (s sortableSet[T]) Less(i, j int) bool {
	return less[int](s[i].index, s[j].index)
}

func (s sortableSet[T]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortableSet[T]) List() []T {
	res := make([]T, s.Len())
	for idx, item := range s {
		res[idx] = item.item
	}
	return res
}

type IPNet struct {
	*net.IPNet
	raw string
}

func (n *IPNet) UnmarshalJSON(raw []byte) (err error) {
	if err = json.Unmarshal(raw, &n.raw); err != nil {
		return err
	}
	if !strings.ContainsRune(n.raw, '/') {
		n.raw = n.raw + "/32"
	}
	if _, n.IPNet, err = net.ParseCIDR(n.raw); err != nil {
		return err
	}
	return nil
}

func (n IPNet) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.raw)
}

func ParseIPNet(netStr string) (IPNet, error) {
	if !strings.ContainsRune(netStr, '/') {
		netStr = netStr + "/32"
	}
	_, ipNet, err := net.ParseCIDR(netStr)
	return IPNet{IPNet: ipNet, raw: netStr}, err
}

type IPNets []IPNet

func (s IPNets) Contains(ip net.IP) bool {
	for _, ipNet := range s {
		if ipNet.IPNet.Contains(ip) {
			return true
		}
	}
	return false
}
