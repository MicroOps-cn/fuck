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
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestLimitWriter(t *testing.T) {
	tests := []struct {
		name        string
		options     []LimitWriterOption
		ignoreError bool
	}{{
		name:        "Test random bytes",
		ignoreError: true,
	}, {
		name:        "Test random bytes with ignore error",
		options:     []LimitWriterOption{LimitWriterIgnoreError},
		ignoreError: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 2<<5; i++ {
				writeCount := 0
				w := &bytes.Buffer{}
				lw := LimitWriter(w, int64(i), tt.options...)
				raw := make([]byte, rand.Intn(1024*1024))
				rand.Read(raw)
				for {
					n, err := lw.Write(raw[writeCount:])
					writeCount += n
					if err != nil {
						if !tt.ignoreError {
							require.NoError(t, err)
							break
						}
						break
					}
					if n == 0 {
						break
					}
				}
				var wantWrite string
				gotW := w.String()
				if i > len(raw) {
					wantWrite = string(raw)
				} else {
					wantWrite = string(raw[:i])
				}

				wantWriteLength := len(raw)
				if tt.ignoreError {
					wantWriteLength = len(wantWrite)
				}
				require.Equalf(t, writeCount, wantWriteLength, "Write length does not match expected value.")
				require.Equal(t, gotW, wantWrite)
			}
		})
	}
}
