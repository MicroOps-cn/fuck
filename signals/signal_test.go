package signals

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestGetSignalHandler(t *testing.T) {
	logger := kitlog.NewNopLogger()
	func() {
		defer func() {
			if r := recover(); r != nil {
				require.Equalf(t, r, ErrorNoInit, "The abnormal information does not match, it should be %s", ErrorNoInit)
			} else {
				t.Error("An exception should be thrown because the SignalHandler is not initialized")
			}
		}()
		SignalHandler()
	}()
	sh := SetupSignalHandler(logger)
	sh2 := SignalHandler()
	sh3 := SetupSignalHandler(logger)
	require.Equalf(t, sh, sh2, "SignalHandler should be a singleton and pointers should not change.")
	require.Equalf(t, sh, sh3, "SignalHandler should be a singleton and pointers should not change.")
}

func TestSetupSignalHandler(t *testing.T) {
	logger := kitlog.NewNopLogger()
	sh := SetupSignalHandler(logger)
	start := time.Now()
	var c []string
	{
		sh.Add(6)
		go func() {
			for i := 0; i < 6; i++ {
				sh.WaitRequest()
				c = append(c, fmt.Sprintf("root:%d", i))
				time.Sleep(time.Second / 3)
				sh.Done()
			}
		}()
	}
	{
		sh.AddRequest(3)
		go func() {
			for i := 0; i < 3; i++ {
				c = append(c, fmt.Sprintf("req:%d", i))
				time.Sleep(time.Second)
				sh.DoneRequest()
			}
		}()
	}
	{
		rand.Intn(10)
		sh.AddFor(3, 5)
		go func() {
			for i := 0; i < 5; i++ {
				sh.WaitRequest()
				c = append(c, fmt.Sprintf("3:%d", i))
				time.Sleep(time.Second)
				sh.DoneFor(3)
			}
		}()
	}
	sh.Wait()
	end := time.Now()
	dur := end.Sub(start)
	require.Equal(t, c[:3], []string{"req:0", "req:1", "req:2"})
	if dur < time.Second*5 || dur > time.Second*51/10 {
		t.Error("Abnormal duration")
	}
}

func TestSetupSignalHandler2(t *testing.T) {
	logger := kitlog.NewNopLogger()
	sh := SetupSignalHandler(logger)
	var c []string
	start := time.Now()
	{
		sh.AddRequest(6)
		go func() {
			for i := 0; i < 6; i++ {
				c = append(c, fmt.Sprintf("req:%d", i))
				time.Sleep(time.Second / 3)
				sh.DoneRequest()
			}
		}()
	}
	{
		for i := 0; i < 3; i++ {
			sh.PreStop(5, func() {
				c = append(c, fmt.Sprintf("5:%d", i))
				time.Sleep(time.Second)
			})
		}
	}
	{
		for i := 0; i < 2; i++ {
			sh.PreStop(LevelRoot, func() {
				c = append(c, fmt.Sprintf("root:%d", i))
				time.Sleep(time.Second)
			})
		}
	}
	{
		sh.PreStop(LevelRoot, func() {
			fmt.Println(c)
		})
	}
	time.Sleep(time.Second)
	sh.safeStop(logger, time.Second*30, func(i int) {
		require.Equal(t, i, 0)
	})
	end := time.Now()
	dur := end.Sub(start)
	require.Equal(t, c, []string{"req:0", "req:1", "req:2", "req:3", "req:4", "req:5", "5:3", "5:3", "5:3", "root:2", "root:2"})
	if dur < time.Second*4 || dur > time.Second*41/10 {
		t.Error("Abnormal duration")
	}
}
