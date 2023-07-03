/*
 Copyright © 2022 MicroOps-cn.

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
	"io"
	"strings"

	"github.com/go-kit/log"
)

type WriterAdapter struct {
	l               log.Logger
	msgKey          string
	prefix          string
	joinPrefixToMsg bool
}

func (a WriterAdapter) Write(p []byte) (n int, err error) {
	a.l.Log(a.msgKey, a.handleMessagePrefix(strings.TrimSuffix(string(p), "\n")))
	return len(p), nil
}

func (a WriterAdapter) handleMessagePrefix(msg string) string {
	if a.prefix == "" {
		return msg
	}

	msg = strings.TrimPrefix(msg, a.prefix)
	if a.joinPrefixToMsg {
		msg = a.prefix + msg
	}
	return msg
}

func MessageKey(key string) WriterAdapterOption {
	return func(a *WriterAdapter) { a.msgKey = key }
}

func Prefix(prefix string, joinPrefixToMsg bool) WriterAdapterOption {
	return func(a *WriterAdapter) { a.prefix = prefix; a.joinPrefixToMsg = joinPrefixToMsg }
}

type WriterAdapterOption func(*WriterAdapter)

func NewWriterAdapter(logger log.Logger, options ...WriterAdapterOption) io.Writer {
	adapter := &WriterAdapter{l: logger, msgKey: "msg"}
	for _, option := range options {
		option(adapter)
	}
	return adapter
}
