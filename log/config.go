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
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
)

// AllowedLevel is a settable identifier for the minimum level a log entry
// must be have.
type AllowedLevel string

func (l AllowedLevel) getOption() level.Option {
	switch l {
	case LevelDebug:
		return level.AllowDebug()
	case LevelInfo:
		return level.AllowInfo()
	case LevelWarn:
		return level.AllowWarn()
	case LevelError:
		return level.AllowError()
	default:
		return level.AllowWarn()
	}
}

func (l *AllowedLevel) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	type plain string
	if err := unmarshal((*plain)(&s)); err != nil {
		return err
	}
	return l.Set(s)
}

func (l AllowedLevel) String() string {
	return string(l)
}

func (l AllowedLevel) Valid() error {
	switch l {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		return nil
	default:
		return errors.Errorf(`unrecognized log level "%s"`, l)
	}
}

// Set updates the value of the allowed level.
func (l *AllowedLevel) Set(s string) error {
	lvl := AllowedLevel(s)
	if len(s) == 0 {
		s = string(LevelWarn)
	}
	if err := lvl.Valid(); err != nil {
		return err
	}
	*l = AllowedLevel(s)
	return nil
}

const (
	LevelDebug AllowedLevel = "debug"
	LevelInfo  AllowedLevel = "info"
	LevelWarn  AllowedLevel = "warn"
	LevelError AllowedLevel = "error"
)

type LoggerCreateFunc func(w io.Writer) log.Logger

var registeredLogFormat sync.Map

func RegisterLogFormat(logFmt AllowedFormat, f LoggerCreateFunc) {
	registeredLogFormat.Store(logFmt, f)
}

func GetRegisteredLogFormats() []AllowedFormat {
	var fmts []AllowedFormat
	registeredLogFormat.Range(func(key, _ any) bool {
		fmts = append(fmts, key.(AllowedFormat))
		return true
	})
	return fmts
}

func init() {
	RegisterLogFormat(FormatLogfmt, log.NewLogfmtLogger)
	RegisterLogFormat(FormatJSON, log.NewJSONLogger)
}

// AllowedFormat is a settable identifier for the output format that the logger can have.
type AllowedFormat string

func (f AllowedFormat) String() string {
	return string(f)
}

func (f AllowedFormat) Valid() error {
	_, ok := registeredLogFormat.Load(f)
	if !ok {
		return errors.Errorf("unrecognized log format %s", f)
	}
	return nil
}

// Set updates the value of the allowed format.
func (f *AllowedFormat) Set(s string) error {
	format := AllowedFormat(s)
	if err := format.Valid(); err != nil {
		return err
	}
	*f = format
	return nil
}

const (
	FormatJSON   AllowedFormat = "json"
	FormatLogfmt AllowedFormat = "logfmt"
)

// Config is a struct containing configurable settings for the logger
type Config struct {
	Level            *AllowedLevel
	Format           *AllowedFormat
	FilePath         string
	FileMaxAge       time.Duration
	FileRotationSize string
	FileRotationTime time.Duration
}

func MustNewConfig(level string, format string) *Config {
	cfg := &Config{Level: new(AllowedLevel), Format: new(AllowedFormat)}
	if err := cfg.Level.Set(level); err != nil {
		panic(err)
	}
	if err := cfg.Format.Set(format); err != nil {
		panic(err)
	}
	return cfg
}
