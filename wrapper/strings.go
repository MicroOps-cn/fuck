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

package w

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Stringer struct {
	stringer func() string
}

func (s Stringer) String() string {
	return s.stringer()
}

func NewStringer(stringer func() string) fmt.Stringer {
	return &Stringer{stringer}
}

func StringEqual(a, b string) bool {
	return a == b
}

type jsonStringer struct {
	v interface{}
}

func (j jsonStringer) String() string {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(j.v); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func JSONStringer(v any) fmt.Stringer {
	return &jsonStringer{v: v}
}
