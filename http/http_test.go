/*
 * Copyright © 2023 MicroOps-cn.
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

package http

import (
	"net/http"
	"testing"

	"github.com/MicroOps-cn/fuck/sets"
	w "github.com/MicroOps-cn/fuck/wrapper"
)

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{{
		name: "",
		args: []string{"/", "aa", "bb", "cc"},
		want: "/aa/bb/cc",
	}, {
		name: "",
		args: []string{"aa", "bb", "cc"},
		want: "/aa/bb/cc",
	}, {
		name: "",
		args: []string{"/aa/", "/bb/"},
		want: "/aa/bb/",
	}, {
		name: "",
		args: []string{"/aa", "/bb/", "cc"},
		want: "/aa/bb/cc",
	}, {
		name: "",
		args: []string{"/aa", "../", "../cc", "dd/ee"},
		want: "/aa/../../cc/dd/ee",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinPath(tt.args...); got != tt.want {
				t.Errorf("JoinPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRemoteAddr(t *testing.T) {
	type args struct {
		r       *http.Request
		trustIp sets.IPNets
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "simple",
		args: args{
			r: &http.Request{Header: map[string][]string{"X-Forwarded-For": {"192.168.1.1,123.222.123.3,192.168.1.1,10.0.0.1,1.1.1.1"}}},
		},
		want: "1.1.1.1",
	}, {
		name: "trustCidr",
		args: args{
			r:       &http.Request{Header: map[string][]string{"X-Forwarded-For": {"192.168.1.1,123.222.123.3,192.168.1.1,10.0.0.1,1.1.1.1"}}},
			trustIp: []sets.IPNet{w.M(sets.ParseIPNet("1.1.1.1"))},
		},
		want: "123.222.123.3",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRemoteAddr(tt.args.r, tt.args.trustIp); got != tt.want {
				t.Errorf("GetRemoteAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}
