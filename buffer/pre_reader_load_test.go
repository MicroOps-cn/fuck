//go:build !make_test

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
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestLoadNewPreReader(t *testing.T) {
	t.Run("Test random data", func(t *testing.T) {
		for i := 0; i < 2<<10; i++ {
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
			require.Equal(t, pr.String(), string(wantPreBuf))
		}
	})
}
