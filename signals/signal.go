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

package signals

import (
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"os"
	"os/signal"
	"sync"
)

type StopChan struct {
	stopCh chan struct{}
	wg     sync.Map
	rootWg sync.WaitGroup
	reqWg  sync.WaitGroup
}

var once = sync.Once{}

const (
	LevelRoot    uint8 = 0
	LevelRequest uint8 = 20
)

func (s *StopChan) WaitRequest() {
	s.reqWg.Wait()
}

func (s *StopChan) DoneRequest() {
	s.reqWg.Done()
}

func (s *StopChan) AddRequest(delta int) {
	s.reqWg.Add(delta)
}

func (s *StopChan) Wait() {
	s.rootWg.Wait()
}

func (s *StopChan) Done() {
	s.rootWg.Done()
}

func (s *StopChan) Add(delta int) {
	s.rootWg.Add(delta)
}

func (s *StopChan) getWaitGroup(level uint8) *sync.WaitGroup {
	wg, _ := s.wg.LoadOrStore(level, &sync.WaitGroup{})
	return wg.(*sync.WaitGroup)
}

func (s *StopChan) WaitFor(level uint8) {
	s.getWaitGroup(level).Wait()
}

func (s *StopChan) DoneFor(level uint8) {
	s.getWaitGroup(level).Done()
}

func (s *StopChan) AddFor(level uint8, delta int) {
	s.getWaitGroup(level).Add(delta)
}

func (s *StopChan) Channel() <-chan struct{} {
	return s.stopCh
}

var stopChan *StopChan

func SetupSignalHandler(logger log.Logger) (stopCh *StopChan) {
	once.Do(func() {
		onlyOneSignalHandler := make(chan struct{})
		close(onlyOneSignalHandler) // panics when called twice
		stopChan = &StopChan{
			stopCh: make(chan struct{}),
		}
		stopChan.wg.Store(LevelRoot, &stopChan.rootWg)
		stopChan.wg.Store(LevelRequest, &stopChan.reqWg)
		c := make(chan os.Signal, 2)
		signal.Notify(c, shutdownSignals...)

		go func() {
			sig := <-c
			level.Info(logger).Log("msg", fmt.Sprintf("收到信号[%s],进程停止.", sig))
			close(stopChan.stopCh)
			stopChan.WaitRequest()
			stopChan.Wait()
			os.Exit(1) // second signal. Exit directly.
		}()
	})
	return stopChan
}

func SignalHandler() (stopCh *StopChan) {
	if stopChan == nil {
		panic("stopChan is not init")
	}
	return stopChan
}
