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
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipeline(t *testing.T) {
	want := []testDataType{{Id: "9"}, {Id: "8"}, {Id: "7"}, {Id: "4"}}
	data := Map[int, testDataType](
		Filter[int](
			[]int{9, -3, 8, 1, 7, 4},
			func(item int) bool { return item > 1 },
		), func(item int) testDataType {
			return testDataType{
				Id: strconv.Itoa(item),
			}
		},
	)
	require.Equal(t, data, want)
}

func TestFilter(t *testing.T) {
	type args struct {
		old []int
		f   func(item int) bool
	}
	tests := []struct {
		name string
		args args
		want []int
	}{{
		name: "int",
		args: args{old: []int{1, 2, 3, 4, 5}, f: func(item int) bool {
			return item >= 4
		}},
		want: []int{4, 5},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filter(tt.args.old, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testDataType struct {
	Id   string
	Name string
}

func TestInclude(t *testing.T) {
	type args struct {
		src    []testDataType
		target testDataType
	}
	tests := []struct {
		name        string
		args        args
		wantInclude bool
		wantHas     bool
	}{{
		name: "Includes struct",
		args: args{
			src:    []testDataType{{Id: "123"}, {Id: "456"}},
			target: testDataType{Id: "456"},
		},
		wantInclude: true,
		wantHas:     true,
	}, {
		name: "Not include struct",
		args: args{
			src:    []testDataType{{Id: "123"}, {Id: "456"}},
			target: testDataType{Id: "aaa"},
		},
		wantInclude: false,
		wantHas:     false,
	}, {
		name: "Includes struct",
		args: args{
			src:    []testDataType{{Id: "123", Name: "HHH"}, {Id: "456", Name: "YYY"}},
			target: testDataType{Id: "456", Name: "YYY"},
		},
		wantInclude: true,
		wantHas:     true,
	}, {
		name: "Not include struct",
		args: args{
			src:    []testDataType{{Id: "123", Name: "HHH"}, {Id: "456", Name: "YYY"}},
			target: testDataType{Id: "123"},
		},
		wantInclude: false,
		wantHas:     true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Include[testDataType](tt.args.src, tt.args.target); !reflect.DeepEqual(got, tt.wantInclude) {
				t.Errorf("Includes() = %v, want %v", got, tt.wantInclude)
			}
			if got := Has[testDataType](tt.args.src, tt.args.target, func(a, b testDataType) bool { return a.Id == b.Id }); !reflect.DeepEqual(got, tt.wantHas) {
				t.Errorf("Has() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestFind(t *testing.T) {
	type plain testDataType
	type args struct {
		s       []plain
		t       plain
		compare func(a, b plain) bool
	}
	tests := []struct {
		name                string
		args                args
		wantFind, wantIndex int
	}{{
		name: "simple find",
		args: args{
			s:       []plain{{Id: "123", Name: "HHH"}, {Id: "456", Name: "YYY"}},
			t:       plain{Id: "123", Name: "HHH"},
			compare: func(a, b plain) bool { return a.Id == b.Id },
		},
		wantFind:  0,
		wantIndex: 0,
	}, {
		name: "simple not find",
		args: args{
			s:       []plain{{Id: "123", Name: "HHH"}, {Id: "456", Name: "YYY"}},
			t:       plain{Id: "456", Name: "YYY"},
			compare: func(a, b plain) bool { return a.Id == b.Id+"1" },
		},
		wantFind:  -1,
		wantIndex: 1,
	}, {
		name: "simple find",
		args: args{
			s:       []plain{{Id: "123", Name: "HHH"}, {Id: "456", Name: "YYY"}},
			t:       plain{Id: "456"},
			compare: func(a, b plain) bool { return a.Id == b.Id },
		},
		wantFind:  1,
		wantIndex: -1,
	}, {
		name: "simple not find",
		args: args{
			s:       []plain{{Id: "123", Name: "HHH"}, {Id: "456", Name: "YYY"}},
			t:       plain{Id: "111"},
			compare: func(a, b plain) bool { return a.Id == b.Id },
		},
		wantFind:  -1,
		wantIndex: -1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Find(tt.args.s, tt.args.t, tt.args.compare); got != tt.wantFind {
				t.Errorf("Find() = %v, want %v", got, tt.wantFind)
			}
			if got := Index(tt.args.s, tt.args.t); got != tt.wantIndex {
				t.Errorf("Find() = %v, want %v", got, tt.wantFind)
			}
		})
	}
}
