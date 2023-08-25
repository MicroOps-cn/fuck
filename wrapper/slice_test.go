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
	"gopkg.in/yaml.v3"
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

func TestToInterface(t *testing.T) {
	ori := []string{"1", "aa", "hello"}
	if got := Interfaces(ori); len(got) != 3 {
		t.Errorf("len(Interfaces()) = %v, want %v", len(got), len(ori))
	}
}

func TestJSONStringer(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "test int",
		args: args{v: 1},
		want: "1",
	}, {
		name: "test float",
		args: args{v: 1.99},
		want: "1.99",
	}, {
		name: "test string",
		args: args{v: "hello"},
		want: `"hello"`,
	}, {
		name: "test struct",
		args: args{v: struct {
			FieldA string  `json:"field_a"`
			FieldB int     `json:"field-b"`
			FieldC float64 `json:"fieldC"`
			FieldD []byte
			fieldE string
		}{FieldA: "hello", FieldB: 123, FieldC: -22.3, FieldD: []byte("world"), fieldE: "bbb"}},
		want: `{"field_a":"hello","field-b":123,"fieldC":-22.3,"FieldD":"d29ybGQ="}`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONStringer(tt.args.v).String(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JSONStringer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLimit(t *testing.T) {
	type args struct {
		s            []byte
		limit        int
		manySuffix   []byte
		hidePosition int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{{
		name: "Test OK - 1",
		args: args{s: []byte("hello"), limit: 10},
		want: []byte("hello"),
	}, {
		name: "Test OK - 2",
		args: args{s: []byte("hello world!!!!!!!!!!!!!"), limit: 10},
		want: []byte("hello worl"),
	}, {
		name: "Test OK - 2",
		args: args{s: []byte("hello world!!!!!!!!!!!!!"), limit: 10, manySuffix: []byte(" ...")},
		want: []byte("hello  ..."),
	}, {
		name: "Test OK - 3",
		args: args{s: []byte("helloworld"), limit: 10},
		want: []byte("helloworld"),
	}, {
		name: "Test OK - 4",
		args: args{
			s:            []byte(`level=info ts=2022-06-02T01:06:10.469Z caller=tls_config.go:191 msg="TLS is disabled." http2=false`),
			limit:        20,
			manySuffix:   []byte(" ... "),
			hidePosition: PosCenter,
		},
		want: []byte("level=i ... p2=false"),
	}, {
		name: "Test OK - 4",
		args: args{
			s:            []byte(`level=info ts=2022-06-02T01:06:10.469Z caller=tls_config.go:191 msg="TLS is disabled." http2=false`),
			limit:        20,
			manySuffix:   []byte(" ... "),
			hidePosition: PosLeft,
		},
		want: []byte(` ... d." http2=false`),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var oldData []byte
			oldData = append(oldData, tt.args.s...)
			got := Limit(tt.args.s, tt.args.limit, tt.args.hidePosition, tt.args.manySuffix...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Limit() = %v, want %v", string(got), string(tt.want))
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("old data = %v, want %v", string(oldData), string(tt.args.s))
			}
		})
	}
}

func TestOneOrMoreStringYamlUnmarshal(t *testing.T) {
	type args struct {
		value string
		obj   OneOrMore[string]
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      OneOrMore[string]
		wantValue string
	}{
		{name: "int string", args: args{value: "123", obj: OneOrMore[string]{}}, want: OneOrMore[string]{"123"}, wantValue: "\"123\"\n"},
		{name: "string", args: args{value: "hello go", obj: OneOrMore[string]{}}, want: OneOrMore[string]{"hello go"}, wantValue: "hello go\n"},
		{name: "strings", args: args{value: "- 123\n- 'hello'", obj: OneOrMore[string]{}}, want: OneOrMore[string]{"123", "hello"}, wantValue: "- \"123\"\n- hello\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := yaml.Unmarshal([]byte(tt.args.value), &tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Unmarshal failed")
			}
			require.Equal(t, tt.want, tt.args.obj)
			data, err := yaml.Marshal(tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Marshal failed")
			}
			require.Equal(t, tt.wantValue, string(data))
		})
	}
}

func TestOneOrMoreInt64YamlUnmarshal(t *testing.T) {
	type args struct {
		value string
		obj   OneOrMore[int64]
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      OneOrMore[int64]
		wantValue string
	}{
		{name: "int64", args: args{value: "123", obj: OneOrMore[int64]{}}, want: OneOrMore[int64]{123}, wantValue: "123\n"},
		{name: "int64s", args: args{value: "- 123\n- 456", obj: OneOrMore[int64]{}}, want: OneOrMore[int64]{123, 456}, wantValue: "- 123\n- 456\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := yaml.Unmarshal([]byte(tt.args.value), &tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Unmarshal failed")
			}
			require.Equal(t, tt.want, tt.args.obj)
			data, err := yaml.Marshal(tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Marshal failed")
			}
			require.Equal(t, tt.wantValue, string(data))
		})
	}
}

func TestOneOrMoreStringJSONUnmarshal(t *testing.T) {
	type args struct {
		value string
		obj   OneOrMore[string]
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      OneOrMore[string]
		wantValue string
	}{
		{name: "int string", args: args{value: `"123"`, obj: OneOrMore[string]{}}, want: OneOrMore[string]{"123"}},
		{name: "string", args: args{value: `"hello go"`, obj: OneOrMore[string]{}}, want: OneOrMore[string]{"hello go"}},
		{name: "strings", args: args{value: `["123","hello"]`, obj: OneOrMore[string]{}}, want: OneOrMore[string]{"123", "hello"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.args.value), &tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Unmarshal failed")
			}
			require.Equal(t, tt.want, tt.args.obj)
			data, err := json.Marshal(tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Marshal failed")
			}
			if len(tt.wantValue) != 0 {
				require.Equal(t, tt.wantValue, string(data))
			} else {
				require.Equal(t, tt.args.value, string(data))
			}
		})
	}
}

func TestOneOrMoreInt64JSONUnmarshal(t *testing.T) {
	type args struct {
		value string
		obj   OneOrMore[int64]
	}
	tests := []struct {
		name      string
		args      args
		wantError bool
		want      OneOrMore[int64]
		wantValue string
	}{
		{name: "int64", args: args{value: "123", obj: OneOrMore[int64]{}}, want: OneOrMore[int64]{123}},
		{name: "int64s", args: args{value: "[123,456]", obj: OneOrMore[int64]{}}, want: OneOrMore[int64]{123, 456}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(tt.args.value), &tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Unmarshal failed")
			}
			require.Equal(t, tt.want, tt.args.obj)
			data, err := json.Marshal(tt.args.obj)
			if err != nil && !tt.wantError {
				require.NoErrorf(t, err, "yaml.Marshal failed")
			}
			if len(tt.wantValue) != 0 {
				require.Equal(t, tt.wantValue, string(data))
			} else {
				require.Equal(t, tt.args.value, string(data))
			}
		})
	}
}
