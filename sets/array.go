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

package sets

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

type ArraySet[T any] struct {
	arr          []T
	cur          int64
	size         int64
	fullCallback func(arr []T, length int64) error
}

func NewArraySet[T any](size int64, val ...T) (*ArraySet[T], error) {
	s := ArraySet[T]{
		arr:  make([]T, size),
		size: size,
		cur:  -1,
	}
	return &s, s.Append(val...)
}

func (s *ArraySet[T]) SetFullCallback(call func([]T, int64) error) {
	s.fullCallback = call
}

func (s *ArraySet[T]) CallFullCallback() error {
	if s.fullCallback != nil {
		return s.fullCallback(s.arr, s.cur+1)
	}
	return nil
}

func (s *ArraySet[T]) Reset() {
	s.arr = make([]T, s.size)
	s.cur = -1
}

var ErrorSetFull = fmt.Errorf("set Is Full")

func (s *ArraySet[T]) Append(values ...T) error {
loop:
	for _, value := range values {
		for _, val := range s.arr {
			if compare(val, value) {
				continue loop
			}
		}
		idx := atomic.AddInt64(&s.cur, 1)
		if idx >= s.size {
			idx = atomic.AddInt64(&s.cur, -1)
			if s.fullCallback != nil {
				if err := s.fullCallback(s.arr, s.cur+1); err != nil {
					return err
				} else if idx = atomic.AddInt64(&s.cur, 1); idx >= s.size {
					idx = atomic.AddInt64(&s.cur, -1)
					return nil
				}
			} else {
				return ErrorSetFull
			}
		}
		s.arr[idx] = value
	}
	return nil
}
func (s *ArraySet[T]) Index(value T) int {
	for idx, val := range s.arr {
		if compare(val, value) {
			return idx
		}
	}
	return -1
}
func (s *ArraySet[T]) Include(value T) bool {
	for _, val := range s.arr {
		if compare(val, value) {
			return true
		}
	}
	return false
}

func (s *ArraySet[T]) Size() int64 {
	return s.size
}

func (s *ArraySet[T]) Length() int64 {
	return s.cur + 1
}

func (s *ArraySet[T]) List() []T {
	return s.arr[:s.cur+1]
}

func compare(a interface{}, b interface{}) bool {
	switch a.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string, bool:
		switch b.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64, string, bool:
			return a == b
		}
	}
	if reflect.TypeOf(a).Kind() == reflect.TypeOf(b).Kind() {
		return reflect.DeepEqual(a, b)
	}
	return false
}
