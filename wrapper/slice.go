//go:build go1.18

/*
 * Copyright © 2022 MicroOps-cn.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package w

import (
	"encoding/json"
	"fmt"
	"reflect"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

func Filter[T any](old []T, f func(item T) bool) []T {
	newSli := make([]T, 0, len(old))
	for _, item := range old {
		if f(item) {
			newSli = append(newSli, item)
		}
	}
	return newSli
}

func Map[S any, T any](old []S, f func(item S) T) []T {
	newSli := make([]T, len(old))
	for idx, item := range old {
		newSli[idx] = f(item)
	}
	return newSli
}

func Has[T any](s []T, t T, compare func(a, b T) bool) bool {
	for _, val := range s {
		if ok := compare(val, t); ok {
			return true
		}
	}
	return false
}

func Include[T comparable](s []T, t T) bool {
	for _, val := range s {
		if val == t {
			return true
		}
	}
	return false
}

func Index[T comparable](s []T, t T) int {
	return slices.Index(s, t)
}

func Find[T any](s []T, t T, compare func(a, b T) bool) int {
	for idx, val := range s {
		if compare(val, t) {
			return idx
		}
	}
	return -1
}

func Interfaces[T any](objs []T) []interface{} {
	newObjs := make([]interface{}, len(objs))
	for idx, obj := range objs {
		newObjs[idx] = obj
	}
	return newObjs
}

const (
	PosRight = iota
	PosLeft
	PosCenter
)

func Limit[T any](s []T, limit int, hidePosition int, manySuffix ...T) []T {
	limit = limit - len(manySuffix)
	if len(s) > limit {
		ret := make([]T, 0, limit+len(manySuffix))
		switch hidePosition {
		case PosRight:
			ret = append(ret, s[:limit]...)
			ret = append(ret, manySuffix...)
			return ret
		case PosLeft:
			ret = append(ret, manySuffix...)
			ret = append(ret, s[len(s)-limit:]...)
			return ret
		case PosCenter:
			ret = append(ret, s[:limit/2]...)
			ret = append(ret, manySuffix...)
			ret = append(ret, s[len(s)-(limit-limit/2):]...)
			return ret
		}
	}
	return s
}

// OneOrMore represents a value that can either be a string
// or an array of strings. Mainly here for serialization purposes
type OneOrMore[T any] []T

func (s OneOrMore[T]) MarshalYAML() (interface{}, error) {
	if len(s) == 1 {
		return (s)[0], nil
	}
	return []T(s), nil

}

func (s *OneOrMore[T]) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		type plain []T
		return value.Decode((*plain)(s))
	case yaml.ScalarNode:
		var c T
		if err := value.Decode(&c); err != nil {
			return err
		}
		*s = OneOrMore[T]{c}
		return nil
	case yaml.AliasNode:
		return value.Alias.Decode(s)
	}
	return fmt.Errorf("unknown value type: %s", value.Tag)
}

type Equal[T any] interface {
	Equal(T) bool
}

// Contains returns true when the value is contained in the slice
func (s OneOrMore[T]) Contains(value T) bool {
	for _, item := range s {
		if v, ok := interface{}(value).(Equal[T]); ok {
			return v.Equal(item)
		}
		if reflect.ValueOf(item).Equal(reflect.ValueOf(value)) {
			return true
		}
	}
	return false
}

// UnmarshalJSON unmarshals this string or array object from a JSON array or signal JSON value
func (s *OneOrMore[T]) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}

	if first == '[' {
		var parsed []T
		if err := json.Unmarshal(data, &parsed); err != nil {
			return err
		}
		*s = OneOrMore[T](parsed)
		return nil
	}

	var single T
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	switch v := interface{}(single).(type) {
	case T:
		*s = OneOrMore[T]([]T{v})
		return nil
	default:
		return fmt.Errorf("only string or array is allowed, not %T", single)
	}
}

// MarshalJSON converts this string or array to a JSON array or JSON string
func (s OneOrMore[T]) MarshalJSON() ([]byte, error) {
	if len(s) == 1 {
		return json.Marshal(s[0])
	}
	return json.Marshal([]T(s))
}
