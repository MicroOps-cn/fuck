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
	"errors"
	"github.com/MicroOps-cn/fuck/signals"
	"github.com/oklog/run"
)

type Group run.Group

var ErrStopping = errors.New("program stopping")

func (g *Group) Add(execute func() error, interrupt func(error)) {
	stopCh := signals.SignalHandler()
	stopCh.Add(1)
	(*run.Group)(g).Add(execute, func(err error) {
		<-stopCh.Channel()
		interrupt(err)
		stopCh.Done()
	})
}

func (g *Group) Run() error {
	stopCh := signals.SignalHandler()
	(*run.Group)(g).Add(func() error {
		<-stopCh.Channel()
		return ErrStopping
	}, func(err error) {})
	return (*run.Group)(g).Run()
}
