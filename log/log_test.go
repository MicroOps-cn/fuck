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

package log

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/require"
)

// Make sure creating and using a testLogger with an empty configuration doesn't
// result in a panic.
func TestDefaultConfig(t *testing.T) {
	testLogger := New()
	if err := testLogger.Log("hello", "world"); err != nil {
		t.Fatal(err)
	}
}

type recordKeyvalLogger struct {
	count int
}

func (r *recordKeyvalLogger) Log(keyvals ...interface{}) error {
	for _, v := range keyvals {
		if fmt.Sprintf("%v", v) == "Log level changed" {
			return nil
		}
	}
	r.count++
	return nil
}

func TestDynamic(t *testing.T) {
	testLogger := newDynamic(&Config{})

	debugLevel := new(AllowedLevel)
	if err := debugLevel.Set("debug"); err != nil {
		t.Fatal(err)
	}
	infoLevel := new(AllowedLevel)
	if err := infoLevel.Set("info"); err != nil {
		t.Fatal(err)
	}

	recorder := &recordKeyvalLogger{}
	testLogger.base = recorder
	testLogger.SetLevel(debugLevel)
	if err := level.Debug(testLogger).Log("hello", "world"); err != nil {
		t.Fatal(err)
	}
	if recorder.count != 1 {
		t.Fatal("log not found")
	}

	recorder.count = 0
	testLogger.SetLevel(infoLevel)
	if err := level.Debug(testLogger).Log("hello", "world"); err != nil {
		t.Fatal(err)
	}
	if recorder.count != 0 {
		t.Fatal("log found")
	}
	if err := level.Info(testLogger).Log("hello", "world"); err != nil {
		t.Fatal(err)
	}
	if recorder.count != 1 {
		t.Fatal("log not found")
	}
	if err := level.Debug(testLogger).Log("hello", "world"); err != nil {
		t.Fatal(err)
	}
	if recorder.count != 1 {
		t.Fatal("extra log found")
	}
}

func TestGetSourceCodeDir(t *testing.T) {
	require.Equal(t, sourceDir, GetSourceCodeDir("log/log_test.go"))
	SetSourceCodeDir("/")
	require.Equal(t, sourceDir, "/")
	SetSourceCodeDir(GetSourceCodeDir("log/log_test.go"))
	require.Equal(t, sourceDir, GetSourceCodeDir("log/log_test.go"))
}

func TestNewTraceLogger(t *testing.T) {
	testLogger := New()
	ctx := context.Background()
	ctx, testTraceLogger := NewContextLogger(ctx, WithLogger(testLogger))
	require.NotNil(t, testTraceLogger)
	traceId := GetContextLogger(ctx)
	require.NotEmpty(t, traceId)
}
