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

package log

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdapterMessageKey(t *testing.T) {
	var w bytes.Buffer
	testLogger := New(&Config{}, WithWriter(&w))
	{
		_, err := NewWriterAdapter(testLogger, MessageKey("qwerty")).Write([]byte("zxcv"))
		require.NoError(t, err)
		ok := strings.HasSuffix(w.String(), "qwerty=zxcv\n")
		require.Truef(t, ok, "The log output is inconsistent with the expected: log=%s", w.String())
	}
}

func TestAdapterPrefix(t *testing.T) {
	type args struct {
		prefix          string
		joinPrefixToMsg bool
		msg             string
	}
	tests := []struct {
		name                string
		args                args
		wantOutputLogSuffix string
	}{
		{name: "Test don't join prefix to msg and msg not container prefix", args: args{joinPrefixToMsg: false, prefix: "qwerty:", msg: "abc"}, wantOutputLogSuffix: " msg=abc\n"},
		{name: "Test join prefix to msg and msg not container prefix", args: args{joinPrefixToMsg: true, prefix: "qwerty:", msg: "abc"}, wantOutputLogSuffix: " msg=qwerty:abc\n"},
		{name: "Test don't join prefix to msg and msg container prefix", args: args{joinPrefixToMsg: false, prefix: "qwerty:", msg: "qwerty:abc"}, wantOutputLogSuffix: " msg=abc\n"},
		{name: "Test join prefix to msg and msg container prefix", args: args{joinPrefixToMsg: true, prefix: "qwerty:", msg: "qwerty:abc"}, wantOutputLogSuffix: " msg=qwerty:abc\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer
			testLogger := New(&Config{}, WithWriter(&w))
			_, err := NewWriterAdapter(testLogger, Prefix(tt.args.prefix, tt.args.joinPrefixToMsg)).Write([]byte(tt.args.msg))
			require.NoError(t, err)
			ok := strings.HasSuffix(w.String(), tt.wantOutputLogSuffix)
			require.Truef(t, ok, "The log output is inconsistent with the expected: log=%s,wantSuffix=%s", w.String(), tt.wantOutputLogSuffix)
		})
	}
}
