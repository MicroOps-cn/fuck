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

package capacity

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseCapacities(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name       string
		args       args
		want       Capacities
		wantErr    bool
		wantString string
	}{{
		name:       "Simple B",
		args:       args{s: "3B"},
		want:       Capacities(3),
		wantString: "3B",
	}, {
		name:       "negative number",
		args:       args{s: "-3KB"},
		want:       Capacities(-3 * 1024),
		wantString: "-3KB",
	}, {
		name:       "Simple MB",
		args:       args{s: "3MB"},
		want:       Capacities(3145728),
		wantString: "3MB",
	}, {
		name:       "Decimal MB",
		args:       args{s: "1.5MB"},
		want:       Capacities(1.5 * 1024 * 1024),
		wantString: "1MB512KB",
	}, {
		name:       "Simple GB",
		args:       args{s: "3GB"},
		want:       Capacities(3 * 1024 * 1024 * 1024),
		wantString: "3GB",
	}, {
		name:       "Decimal GB",
		args:       args{s: "0.6GB"},
		want:       Capacities(6 * 1024 * 1024 * 1024 / 10),
		wantString: "614MB409KB614B",
	}, {
		name:       "compound",
		args:       args{s: "-1gb1M2.3MB5b"},
		want:       Capacities(-1077202129),
		wantString: "-1GB3MB307KB209B",
	}, {
		name:    "out of range",
		args:    args{s: "10000000T10gb1M2.3MB5b"},
		wantErr: true,
	}, {
		name:    "out of range 2",
		args:    args{s: "1T10gb1M2.1111111111111111111111"},
		wantErr: true,
	}, {
		name:       "out of range 2",
		args:       args{s: "1.9999999999999999999999999999999999999999999999T"},
		want:       Capacities(2199023255552),
		wantString: "2TB",
	}, {
		name:    "not unit",
		args:    args{s: "1.9999999999999999999999999999999999999999999999"},
		wantErr: true,
	}, {
		name:    "out of range 3",
		args:    args{s: "9999999999999999999999999999999999999999999999.1T"},
		wantErr: true,
	}, {
		name:    "Null",
		args:    args{s: ""},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCapacities(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCapacities() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("ParseCapacities() got = %v(%d), want %s(%d)", got, got, tt.want.String(), tt.want)
				return
			}
			gotString := got.String()
			if gotString != tt.wantString {
				t.Errorf("ParseCapacities() got = %s, want %s", gotString, tt.args.s)
				return
			}
			{
				var c Capacities
				err = json.Unmarshal([]byte(`"`+tt.args.s+`"`), &c)
				if (err != nil) != tt.wantErr {
					t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if c != tt.want {
					t.Errorf("json.Unmarshal() got = %v(%d), want %s(%d)", got, got, tt.want.String(), tt.want)
				}
				if jsonString, err := json.Marshal(c); err != nil {
					require.NoError(t, err)
				} else if strings.TrimSpace(string(jsonString)) != fmt.Sprintf(`"%s"`, tt.wantString) {
					t.Errorf("json.Marshal() got = %s, want %s", jsonString, tt.args.s)
				} else {
					fmt.Println(string(jsonString))
				}
			}

			{
				var c Capacities
				err = yaml.Unmarshal([]byte(`"`+tt.args.s+`"`), &c)
				if (err != nil) != tt.wantErr {
					t.Errorf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if c != tt.want {
					t.Errorf("yaml.Unmarshal() got = %v(%d), want %s(%d)", got, got, tt.want.String(), tt.want)
				}
				if yamlString, err := yaml.Marshal(c); err != nil {
					require.NoError(t, err)
				} else if strings.TrimSpace(string(yamlString)) != tt.wantString {
					t.Errorf("yaml.Marshal() got = %s, want %s", yamlString, tt.args.s)
				}
			}
		})
	}
}
