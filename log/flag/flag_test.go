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
	"flag"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/MicroOps-cn/fuck/log"
)

type flagParser interface {
	Parse(arguments []string) error
}

type flagParserSet struct {
	set FlagSet
	p   func(arguments []string) error
}

func (f flagParserSet) Parse(arguments []string) error {
	return f.p(arguments)
}

func (f flagParserSet) StringVar(p *string, name string, value string, usage string) {
	f.set.StringVar(p, name, value, usage)
}

func (f flagParserSet) DurationVar(p *time.Duration, name string, value time.Duration, usage string) {
	f.set.DurationVar(p, name, value, usage)
}

type flagParseSet interface {
	flagParser
	FlagSet
}

func newFlagParserSet(set FlagSet, p func(arguments []string) error) flagParseSet {
	return &flagParserSet{p: p, set: set}
}

func TestAddFlags(t *testing.T) {
	kgCmd := kingpin.New("test", "")
	type args struct {
		set interface {
			flagParser
			FlagSet
		}
		config *log.Config
	}
	tests := []struct {
		name  string
		args  args
		parse func()
	}{{
		name: "test std flags",
		args: args{
			set:    flag.NewFlagSet("test", flag.PanicOnError),
			config: &log.Config{},
		},
	}, {
		name: "test spf13/pflag flags",
		args: args{
			set:    pflag.NewFlagSet("test", pflag.PanicOnError),
			config: &log.Config{},
		},
	}, {
		name: "test alecthomas/kingpin.v2 flags",
		args: args{
			set: newFlagParserSet(NewHandler(func(p *string, name string, value string, usage string) {
				kgCmd.Flag(name, usage).Default(value).StringVar(p)
			}, func(p *time.Duration, name string, value time.Duration, usage string) {
				kgCmd.Flag(name, usage).Default(value.String()).DurationVar(p)
			}), func(arguments []string) error {
				_, err := kgCmd.Parse(arguments)
				return err
			}),
			config: &log.Config{},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				AddFlags(tt.args.set, tt.args.config)
				err := tt.args.set.(flagParser).Parse([]string{})
				require.NoError(t, err)
				require.Equal(t, tt.args.config.Level.String(), "info")
				require.Equal(t, tt.args.config.Format.String(), "logfmt")
			})
			t.Run("has args", func(t *testing.T) {
				err := tt.args.set.(flagParser).Parse([]string{"--log.level=error", "--log.format=json"})
				require.NoError(t, err)
				require.Equal(t, tt.args.config.Level.String(), "error")
				require.Equal(t, tt.args.config.Format.String(), "json")
			})
		})
	}
}
