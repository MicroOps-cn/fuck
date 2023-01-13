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

package log

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUnmarshallLevel(t *testing.T) {
	l := new(AllowedLevel)
	err := yaml.Unmarshal([]byte(`debug`), l)
	if err != nil {
		t.Error(err)
	}
	if *l != "debug" {
		t.Errorf("expected %s, got %s", "debug", *l)
	}
}

func TestUnmarshallEmptyLevel(t *testing.T) {
	l := new(AllowedLevel)
	err := yaml.Unmarshal([]byte(``), l)
	if err != nil {
		t.Error(err)
	}
	if *l != "" {
		t.Errorf("expected empty level, got %s", *l)
	}
}

func TestUnmarshallBadLevel(t *testing.T) {
	l := new(AllowedLevel)
	err := yaml.Unmarshal([]byte(`debugg`), l)
	if err == nil {
		t.Error("expected error")
	}
	expErr := `unrecognized log level "debugg"`
	if err.Error() != expErr {
		t.Errorf("expected error %s, got %s", expErr, err.Error())
	}
	if *l != "" {
		t.Errorf("expected empty level, got %s", *l)
	}
}

func TestAllowedLevel_UnmarshalYAML(t *testing.T) {
	type args struct {
		level string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    AllowedLevel
	}{
		{name: "test debug", args: args{level: "debug"}, want: LevelDebug, wantErr: false},
		{name: "test info", args: args{level: "info"}, want: LevelInfo, wantErr: false},
		{name: "test warn", args: args{level: "warn"}, want: LevelWarn, wantErr: false},
		{name: "test error", args: args{level: "error"}, want: LevelError, wantErr: false},
		{name: "test empty level", args: args{level: ""}, want: AllowedLevel(""), wantErr: false},
		{name: "test bad level", args: args{level: "debugg"}, want: AllowedLevel(""), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var level AllowedLevel
			if err := yaml.Unmarshal([]byte(tt.args.level), &level); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, level)
		})
	}
}

func TestGetRegisteredLogFormats(t *testing.T) {
	formats := GetRegisteredLogFormats()
	var actual []string
	for _, format := range formats {
		actual = append(actual, string(format))
	}
	expected := []string{string(FormatLogfmt), string(FormatJSON)}
	sort.Strings(actual)
	sort.Strings(expected)
	require.Equal(t, expected, actual)
}
