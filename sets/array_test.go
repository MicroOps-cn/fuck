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
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_compare(t *testing.T) {
	type String string
	type args struct {
		a interface{}
		b interface{}
	}
	a := args{}
	b := &a
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "Test string equal", args: args{a: "1234567890-=qwertyuiopasdfghjkl;zxcvbnm,.", b: "1234567890-=qwertyuiopasdfghjkl;zxcvbnm,."}, want: true},
		{name: "Test string not equal", args: args{a: "1234567890-=qwertyuiopasdfghjkl;zxcvbnm,.", b: "1234567890-=qwertyuiopasdfghjkl;zxcvbnm,2"}, want: false},
		{name: "Test String equal", args: args{a: String("1234567890-=qwertyuiopasdfghjkl;zxcvbnm,."), b: String("1234567890-=qwertyuiopasdfghjkl;zxcvbnm,.")}, want: true},
		{name: "Test String not equal", args: args{a: String("1234567890-=qwertyuiopasdfghjkl;zxcvbnm,."), b: String("1234567890-=qwertyuiopasdfghjkl;zxcvbnm,2")}, want: false},
		{name: "Test float32 not equal", args: args{a: float32(0.3000001), b: float32(0.3000000)}, want: false},
		{name: "Test Different types float", args: args{a: float32(0.3000001), b: float64(0.3000001)}, want: false},
		{name: "Test Different types", args: args{a: "s11111111", b: []byte("s11111111")}, want: false},
		{name: "Test ptr and struct", args: args{a: a, b: b}, want: false},
		{name: "Test ptr equal", args: args{a: &a, b: b}, want: true},
		{name: "Test null struct", args: args{a: args{}, b: args{}}, want: true},
		{name: "Test struct equal", args: args{a: args{a: 11}, b: args{a: 11}}, want: true},
		{name: "Test struct not equal", args: args{a: args{a: 11}, b: args{a: 22}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compare(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewArraySet(t *testing.T) {
	s, err := NewArraySet[int](10, 1, 72, 3, 4, 5, 6, 18, 8, 9, 10, 2, 3, 4, 1, 10, 18, 44)
	require.Equal(t, err, ErrorSetFull)
	require.Equal(t, s.List(), []int{1, 72, 3, 4, 5, 6, 18, 8, 9, 10})
	var buf []int
	require.Len(t, s.List(), 10)
	s.SetFullCallback(func(ints []int, i int64) error {
		s.Reset()
		buf = append(buf, ints...)
		return nil
	})
	err = s.Append(1, 2, 3, 4, 5)
	require.NoError(t, err)
	require.Equal(t, s.List(), []int{2, 3, 4, 5})
	require.Len(t, s.List(), 4)

	s.SetFullCallback(func(ints []int, i int64) error { return nil })
	err = s.Append(11, 22, 33, 44, 55, 66, 77, 88, 99, 00, 22) //nolint:gofumpt
	require.NoError(t, err)
	require.Equal(t, s.List(), []int{2, 3, 4, 5, 11, 22, 33, 44, 55, 66})
}
