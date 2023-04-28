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

package http

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/MicroOps-cn/fuck/sets"
)

func joinPath(a, b string) string {
	if len(a) == 0 {
		a = "/"
	} else if a[0] != '/' {
		a = "/" + a
	}

	if len(b) != 0 && b[0] == '/' {
		b = b[1:]
	}

	if len(b) != 0 && len(a) > 1 && a[len(a)-1] != '/' {
		a = a + "/"
	}

	return a + b
}

func JoinPath(p ...string) string {
	if len(p) == 0 {
		return ""
	} else if len(p) == 1 {
		return p[0]
	}
	ret := p[0]
	for i := 1; i < len(p); i++ {
		ret = joinPath(ret, p[i])
	}
	return ret
}

func GetRemoteAddr(r *http.Request, trustIp sets.IPNets) string {
	remoteAddr, _, _ := strings.Cut(r.RemoteAddr, ":")
	ipSet := []string{remoteAddr}
	ipSet = append(ipSet, strings.Split(r.Header.Get("X-Forwarded-For"), ",")...)
	for i := len(ipSet) - 1; i > 0; i-- {
		fmt.Println(ipSet[i], net.ParseIP(ipSet[i]))
		if ip := net.ParseIP(ipSet[i]); ip != nil {
			if trustIp.Contains(ip) || !ip.IsGlobalUnicast() || ip.IsPrivate() {
				continue
			}
			return ipSet[i]
		}
	}
	return remoteAddr
}
