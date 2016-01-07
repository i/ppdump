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

	a := NewA(NewAOpts{
		Interval:  time.Second,
		HardLimit: 500,
		Path:      "./",
		Profiles: map[string]int{
			"goroutine": 2,
		},
	})

	a.Start()
	doSomething()
	time.Sleep(time.Second * 5)
	a.Stop()
}

func doSomething() {
	for i := 0; i < 500; i++ {
		go doWork()
	}
}

func doWork() {
	time.Sleep(time.Second * 2)
}
