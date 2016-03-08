package ppdump

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewA(t *testing.T) {
	f, err := os.Create("./ppdump.out")
	require.NoError(t, err)
	defer f.Close()

	d, err := NewDumper(Config{
		Interval: time.Second,
		Writer:   os.Stdout,
		Profiles: map[string]Profile{
			"goroutine": {
				Threshold: 500,
				Debug:     2,
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
