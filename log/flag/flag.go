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

package flag

import (
	"fmt"
	"time"

	"github.com/MicroOps-cn/fuck/log"
)

// LevelFlagName is the canonical flag name to configure the allowed log level
// within Prometheus projects.
const LevelFlagName = "log.level"

// LevelFlagHelp is the help description for the log.level flag.
const LevelFlagHelp = "Only log messages with the given severity or above. One of: [debug, info, warn, error]"

// FormatFlagName is the canonical flag name to configure the log format
// within Prometheus projects.
const FormatFlagName = "log.format"

// FormatFlagHelp is the help description for the log.format flag.
var FormatFlagHelp = func() string {
	return fmt.Sprintf("Output format of log messages. One of: %v", log.GetRegisteredLogFormats())
}

type FlagSet interface {
	StringVar(p *string, name string, value string, usage string)
	DurationVar(p *time.Duration, name string, value time.Duration, usage string)
}

// AddFlags adds the flags used by this package to the Kingpin application.
// To use the default Kingpin application, call AddFlags(kingpin.CommandLine)
func AddFlags(set FlagSet, config *log.Config) {
	if config == nil {
		config = log.DefaultLoggerConfig
	}
	config.Level = new(log.AllowedLevel)
	set.StringVar((*string)(config.Level), LevelFlagName, string(log.LevelInfo), LevelFlagHelp)
	config.Format = new(log.AllowedFormat)
	set.StringVar((*string)(config.Format), FormatFlagName, string(log.FormatLogfmt), FormatFlagHelp())
	set.StringVar(&config.FilePath, "log.file", "/dev/stderr", "The file path used to store logs.")
	set.DurationVar(&config.FileRotationTime, "log.file-rotation-time", time.Hour*24, "Rotation cycle of log file")
	set.DurationVar(&config.FileMaxAge, "log.file-max-age", time.Hour*24*7, "Maximum retention time of log files")
	set.StringVar(&config.FileRotationSize, "log.file-rotation-size", "100m", "Rotation size of log file")
}

type flagHandler struct {
	sVar func(p *string, name string, value string, usage string)
	dVar func(p *time.Duration, name string, value time.Duration, usage string)
}

func (h flagHandler) DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	h.dVar(p, name, value, usage)
}

func (h flagHandler) StringVar(p *string, name string, value string, usage string) {
	h.sVar(p, name, value, usage)
}

func NewHandler(sVar func(p *string, name string, value string, usage string), dVar func(p *time.Duration, name string, value time.Duration, usage string)) FlagSet {
	return &flagHandler{sVar: sVar, dVar: dVar}
}
