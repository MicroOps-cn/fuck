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

package log

import (
	"fmt"
	"io"

	"github.com/go-kit/log"
)

type withPrint struct{}

func WithPrint(raw interface{}) Option {
	return func(l log.Logger) log.Logger {
		return log.With(l, withPrint{}, raw)
	}
}

type extLogger struct {
	logger log.Logger
	w      io.Writer
}

func writeln(w io.Writer, v []byte) {
	if len(v) > 0 {
		_, _ = w.Write(v)
		if v[len(v)-1] != '\n' {
			_, _ = w.Write([]byte{'\n'})
		}
	}
}

func (l extLogger) Log(keyvals ...interface{}) error {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, log.ErrMissingValue)
	}
	var values []interface{}
	callerIndex := -1
	for i := 0; i < len(keyvals); {
		key := keyvals[i]
		val := keyvals[i+1]
		switch key.(type) {
		case callerName:
			if callerIndex < 0 {
				callerIndex = i
			} else {
				keyvals[callerIndex+1] = val
				keyvals = append(keyvals[:i], keyvals[i+2:]...)
				continue
			}
		case withPrint:
			values = append(values, val)
			keyvals = append(keyvals[:i], keyvals[i+2:]...)
			continue
		}
		i += 2
	}
	defer func() {
		for _, value := range values {
			switch v := value.(type) {
			case string:
				writeln(l.w, []byte(v))
			case fmt.Stringer:
				writeln(l.w, []byte(v.String()))
			default:
				if value != nil {
					writeln(l.w, []byte(fmt.Sprintf("%#v", value)))
				}
			}
		}
	}()

	return l.logger.Log(keyvals...)
}
