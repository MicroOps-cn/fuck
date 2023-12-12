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

package w

import (
	"fmt"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	"github.com/MicroOps-cn/fuck/signals"
)

func TestGroup_Run(t *testing.T) {
	ExitFunc = func(code int) {
		require.Equal(t, code, 0)
	}
	signals.SetupSignalHandler(kitlog.NewNopLogger())
	var g Group
	start := time.Now()
	timer1 := time.NewTimer(time.Second * 1)
	g.Add(func() error {
		<-timer1.C
		return fmt.Errorf("auto stop 1")
	}, func(err error) {
		timer1.Reset(0)
		require.ErrorContains(t, err, "auto stop 1")
	})
	timer2 := time.NewTimer(time.Second * 3)
	g.Add(func() error {
		<-timer2.C
		return fmt.Errorf("auto stop 2")
	}, func(err error) {
		require.ErrorContains(t, err, "auto stop 1")
		timer2.Reset(0)
		time.Sleep(time.Second)
	})
	require.ErrorContains(t, g.Run(), "auto stop 1")
	end := time.Now()
	dur := end.Sub(start)
	if dur < time.Second*2 || dur > time.Second*21/10 {
		t.Error("Abnormal duration")
	}
}
