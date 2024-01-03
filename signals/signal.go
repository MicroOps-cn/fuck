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
	"errors"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type stopFunc func()

type stopFuncs []stopFunc

type Handler struct {
	stopCh       chan struct{}
	wg           sync.Map
	preStopFuncs []stopFuncs
	mux          sync.Mutex
	logger       log.Logger
}

var once = sync.Once{}

const (
	LevelRoot    uint8 = 0
	LevelTrace   uint8 = 5
	LevelDB      uint8 = 10
	LevelRequest uint8 = 20
	LevelMax     uint8 = 31
)

func (s *Handler) WaitRequest() {
	s.getWaitGroup(LevelRequest).Wait()
}

func (s *Handler) DoneRequest() {
	s.getWaitGroup(LevelRequest).Done()
}

func (s *Handler) AddRequest(delta int) {
	s.getWaitGroup(LevelRequest).Add(delta)
}

func (s *Handler) Wait() {
	s.getWaitGroup(LevelRoot).Wait()
}

func (s *Handler) Done() {
	s.getWaitGroup(LevelRoot).Done()
}

func (s *Handler) Add(delta int) {
	s.getWaitGroup(LevelRoot).Add(delta)
}

func (s *Handler) getWaitGroup(level uint8) *sync.WaitGroup {
	wg, _ := s.wg.LoadOrStore(level, &sync.WaitGroup{})
	return wg.(*sync.WaitGroup)
}

func (s *Handler) WaitFor(level uint8) {
	s.getWaitGroup(level).Wait()
}

func (s *Handler) DoneFor(level uint8) {
	s.getWaitGroup(level).Done()
}

func (s *Handler) AddFor(level uint8, delta int) {
	s.getWaitGroup(level).Add(delta)
}

func (s *Handler) Channel() <-chan struct{} {
	return s.stopCh
}

func (s *Handler) PreStop(level uint8, f stopFunc) {
	if level > LevelMax || level < LevelRoot {
		panic(ErrorLevelOutOfBounds)
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.preStopFuncs[level] = append(s.preStopFuncs[level], f)
}

func (s *Handler) safeStop(logger log.Logger, timeout time.Duration, exitFunc func(int)) {
	go func() {
		timer := time.NewTimer(timeout)
		<-timer.C
		exitFunc(0)
	}()
	stopHandler.mux.Lock()
	defer stopHandler.mux.Unlock()
	level.Info(logger).Log("msg", "Received stop signal, the process is about to stop.")
	close(stopHandler.stopCh)
	for lvl := len(stopHandler.preStopFuncs) - 1; lvl >= 0; lvl-- {
		funcs := stopHandler.preStopFuncs[lvl]
		var wg sync.WaitGroup
		wg.Add(len(funcs))
		for _, f := range funcs {
			go func(sf stopFunc) {
				defer wg.Done()
				sf()
			}(f)
		}
		wg.Wait()
		stopHandler.WaitFor(uint8(lvl))
	}
	exitFunc(0)
}

func (s *Handler) SafeStop(timeout time.Duration, exitFunc func(int)) {
	s.safeStop(s.logger, timeout, exitFunc)
}

var stopHandler *Handler

func SetupSignalHandler(logger log.Logger) (stopCh *Handler) {
	once.Do(func() {
		onlyOneSignalHandler := make(chan struct{})
		close(onlyOneSignalHandler) // panics when called twice
		stopHandler = &Handler{
			stopCh:       make(chan struct{}),
			preStopFuncs: make([]stopFuncs, LevelMax+1),
			logger:       logger,
		}
		c := make(chan os.Signal, 2)
		signal.Notify(c, shutdownSignals...)

		go func() {
			sig := <-c
			stopHandler.safeStop(log.With(logger, "signal", sig), time.Second*30, os.Exit)
			// second signal. Exit directly.
		}()
	})
	return stopHandler
}

var ErrorNoInit = errors.New("stopChan is not init")
var ErrorLevelOutOfBounds = errors.New("level out of bounds")

func SignalHandler() (stopCh *Handler) {
	if stopHandler == nil {
		panic(ErrorNoInit)
	}
	return stopHandler
}
