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

func M[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func E[T any](_ T, err error) error {
	return err
}

func P[T any](o T) *T {
	return &o
}

func DefaultPointer[T any](vals ...*T) *T {
	for _, val := range vals {
		if val != nil {
			return val
		}
	}
	return nil
}

func DefaultString(vals ...string) string {
	for _, val := range vals {
		if len(val) != 0 {
			return val
		}
	}
	return ""
}

type Number interface {
	~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float64 | ~float32 | ~int | ~uint
}

func DefaultNumber[T Number](vals ...T) T {
	for _, val := range vals {
		if val != 0 {
			return val
		}
	}
	return 0
}

func Pipe[sT any, dT any](v sT, e error) func(func(sT) (dT, error)) (dT, error) {
	return func(f func(sT) (dT, error)) (dT, error) {
		if e != nil {
			return *new(dT), e
		}
		return f(v)
	}
}

func T[vT any](expr bool, trueVal, falseVal vT) vT {
	if expr {
		return trueVal
	}
	return falseVal
}
