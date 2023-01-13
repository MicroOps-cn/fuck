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
	"os"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// NewDynamic returns a new leveled logger. Each logged line will be annotated
// with a timestamp. The output always goes to stderr. Some properties can be
// changed, like the level.
func newDynamic(config *Config) *logger {
	var l log.Logger
	if config.Format != nil && *config.Format == "json" {
		l = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	} else {
		l = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	}
	l = log.With(l, "ts", TimestampFormat, CallerName, DefaultCaller)

	lo := &logger{
		base:    l,
		leveled: l,
	}
	if config.Level != nil {
		lo.SetLevel(config.Level)
	}
	return lo
}

type logger struct {
	base         log.Logger
	leveled      log.Logger
	currentLevel *AllowedLevel
	mtx          sync.Mutex
}

// Log implements logger.Log.
func (l *logger) Log(keyvals ...interface{}) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.leveled.Log(keyvals...)
}

// SetLevel changes the log level.
func (l *logger) SetLevel(lvl *AllowedLevel) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if lvl != nil {
		if l.currentLevel != nil && *l.currentLevel != *lvl {
			_ = l.base.Log("msg", "Log level changed", "prev", l.currentLevel, "current", lvl)
		}
		l.currentLevel = lvl
	}
	l.leveled = level.NewFilter(l.base, lvl.getOption())
}
