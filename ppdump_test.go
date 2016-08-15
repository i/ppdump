package ppdump

import (
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewA(t *testing.T) {
	f, err := os.Create("./ppdump.out")
	require.NoError(t, err)
	defer f.Close()

	d := New(Config{
		PollInterval: time.Second,
		Profiles: map[string]ProfileOpts{
			"goroutine": {
				Threshold: 500,
				Action: func(p *pprof.Profile) {
				},
			},
		},
	})
	require.NoError(t, err)

	d.Start()
	doSomething()
	time.Sleep(time.Second * 5)
	d.Stop()
}

func doSomething() {
	for i := 0; i < 500; i++ {
		go doWork()
	}
}

func doWork() {
	time.Sleep(time.Second * 2)
}
