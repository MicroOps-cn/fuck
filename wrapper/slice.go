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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
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
		if ok := val == t; ok {
			return true
		}
	}
	return false
}

func Index[T comparable](s []T, t T) int {
	for idx, val := range s {
		if ok := val == t; ok {
			return idx
		}
	}
	return -1
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

type jsonStringer struct {
	v interface{}
}

func (j jsonStringer) String() string {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(j.v); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func JSONStringer(v any) fmt.Stringer {
	return &jsonStringer{v: v}
}
