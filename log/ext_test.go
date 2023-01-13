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

	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/require"
)

func TestWithPrint(t *testing.T) {
	var w bytes.Buffer
	testLogger := New(&Config{}, WithWriter(&w))
	{
		withPrintLogger := WithPrint("Hello1")(testLogger)
		withPrintLogger = WithPrint("Hello2")(withPrintLogger)
		if err := level.Error(withPrintLogger).Log("hello", "world"); err != nil {
			t.Fatal(err)
		}
		ok := strings.HasSuffix(w.String(), "hello=world\nHello1\nHello2\n")
		require.Truef(t, ok, "The log output is inconsistent with the expected: log=%s", w.String())
	}
}
