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

package g

import (
	"encoding/binary"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.opentelemetry.io/otel/trace"
)

const midUint64 = uint64(1) << 63

func NewId(seed ...string) string {
	return trace.TraceID(NewUUID(seed...)).String()
}

func NewUUID(seed ...string) uuid.UUID {
	ts := uint64(time.Now().UnixMicro())
	if ts < midUint64 {
		var hash uint64
		for _, s := range seed {
			seedBytes := []byte(s)
			for i := 0; i < len(seedBytes); i += 8 {
				if len(seedBytes[i:]) <= 8 {
					tmp := make([]byte, 8)
					copy(tmp, seedBytes[i:])
					hash += binary.BigEndian.Uint64(tmp)
					break
				}
				hash += binary.BigEndian.Uint64(seedBytes[i : i+8])
			}
		}
		if hash > midUint64 {
			hash = hash - midUint64
		}
		ts += hash
	}

	id := uuid.Must(uuid.NewV4())
	binary.BigEndian.PutUint64(id[:8], ts)
	return id
}
