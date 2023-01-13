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

func (l extLogger) Log(keyvals ...interface{}) error {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, log.ErrMissingValue)
	}
	var values []interface{}

	for i := 0; i < len(keyvals); {
		key := keyvals[i]
		val := keyvals[i+1]
		if _, ok := key.(withPrint); ok {
			values = append(values, val)
			keyvals = append(keyvals[:i], keyvals[i+2:]...)
			continue
		}
		i += 2
	}
	defer func() {
		for _, value := range values {
			if s, ok := value.(string); ok && len(s) > 0 {
				_, _ = l.w.Write([]byte(s))
				if s[len(s)-1] != '\n' {
					_, _ = l.w.Write([]byte{'\n'})
				}
			} else if str, ok := value.(fmt.Stringer); ok {
				s = str.String()
				if len(s) > 0 {
					_, _ = l.w.Write([]byte(s))
					if s[len(s)-1] != '\n' {
						_, _ = l.w.Write([]byte{'\n'})
					}
				}
			} else if value != nil {
				_, _ = fmt.Fprintf(l.w, "%#v\n", value)
			}
		}
	}()
	return l.logger.Log(keyvals...)
}
