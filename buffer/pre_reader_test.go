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

package buffer

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

type testErrorReader struct {
	err error
}

func (t testErrorReader) Read(_ []byte) (n int, err error) {
	return 0, t.err
}

func TestNewPreReader(t *testing.T) {
	t.Run("Test random data", func(t *testing.T) {
		nullDev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		require.NoError(t, err)
		for i := 0; i < 2<<5; i++ {
			raw := make([]byte, rand.Intn(1024*1024))
			rand.Read(raw)
			r := io.NopCloser(bytes.NewBuffer(raw))
			pr, err := NewPreReader(r, i)
			require.NoError(t, err)
			var wantPreBuf []byte
			if len(raw) < i {
				wantPreBuf = raw
			} else {
				wantPreBuf = raw[:i]
			}

			_, err = io.Copy(nullDev, pr)
			require.NoError(t, err)
			require.Equal(t, pr.String(), string(wantPreBuf))
			require.Equal(t, string(pr.Buffer()), string(wantPreBuf))
			err = pr.Close()
			require.NoError(t, err)
		}
	})
	t.Run("Test EOF error", func(t *testing.T) {
		r := io.NopCloser(&testErrorReader{io.EOF})
		pr, err := NewPreReader(r, 1024)
		require.NoError(t, err)
		require.Equal(t, pr.String(), "")
	})
	t.Run("Test error", func(t *testing.T) {
		r := io.NopCloser(&testErrorReader{fmt.Errorf("error: %s", "test error")})
		pr, err := NewPreReader(r, 1024)
		require.Error(t, err)
		require.Equal(t, pr.String(), "")
	})
	t.Run("Test EOF", func(t *testing.T) {
		nullDev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		require.NoError(t, err)
		r := io.NopCloser(bytes.NewBufferString("111"))
		pr, err := NewPreReader(r, 1024)
		require.NoError(t, err)
		require.Equal(t, pr.String(), "111")
		_, err = io.Copy(nullDev, pr)
		require.NoError(t, err)
	})
}
