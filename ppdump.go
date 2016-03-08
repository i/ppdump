package ppdump

import (
	"fmt"
	"io"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"gopkg.in/validator.v2"
)

const (
	DefaultBufferSize = 1000
	DefaultInterval   = time.Second
)

var std *Dumper

type Config struct {
	Interval     time.Duration  `validate:"nonzero"` // how often to poll for goroutines
	Profiles     map[string]int `validate:"nonzero"` // map of profile names to debug level
	Writer       io.Writer      // where to write the dump to
	Throttle     time.Duration  // time to wait before writing another dump
	HardLimit    int            // number of goroutines to trigger a dump
	ThresholdPct float64        // percentage of increase to trigger a dump
}

// Start begins the
func Start(c Config) error {
	var err error
	std, err = NewDumper(c)
	if err != nil {
		return err
	}
	std.Start()
	return nil
}

func Stop() {
	if std != nil {
		std.Stop()
	}
}

type Dumper struct {
	sync.Mutex

	interval time.Duration
	throttle time.Duration
	lim      int
	thr      float64
	avg      float64
	nv       int64
	writer   io.Writer
	profiles map[string]int
	stop     chan struct{}
	lastDump time.Time
}

func NewDumper(c Config) (*Dumper, error) {
	if c.Writer == nil {
		return nil, fmt.Errorf("no writer")
	}
	if err := validator.Validate(c); err != nil {
		return nil, err
	}

	return &Dumper{
		interval: c.Interval,
		throttle: c.Throttle,
		lim:      c.HardLimit,
		thr:      c.ThresholdPct,
		profiles: c.Profiles,
		writer:   c.Writer,
	}, nil
}

func (d *Dumper) Start() {
	d.Lock()
	defer d.Unlock()

	d.stop = make(chan struct{})

	if d.interval == 0 {
		d.interval = DefaultInterval
	}
	go d.runLoop()
}

func (d *Dumper) runLoop() {
	t := time.NewTicker(d.interval)
	for {
		select {
		case <-t.C:
			if d.thresholdExceeded() {
				d.dump()
			}
		case <-d.stop:
			return
		}
	}
}

func (d *Dumper) dump() {
	d.Lock()
	defer d.Unlock()

	now := time.Now()
	if now.Before(d.lastDump.Add(d.throttle)) {
		return
	}
	d.lastDump = now

	for profile, debug := range d.profiles {
		p := pprof.Lookup(profile)
		if p == nil {
			continue
		}

		p.WriteTo(d.writer, debug)
	}
}

// Stops the runloop. No-op if called more than once.
func (d *Dumper) Stop() {
	d.Lock()
	defer d.Unlock()
	defer func() { recover() }() // doesn't matter if the channel is closed twice
	close(d.stop)
}

func (d *Dumper) thresholdExceeded() bool {
	n := runtime.NumGoroutine()
	oldAvg := d.avg
	defer d.updateAvg(n)
	return (d.lim != 0 && runtime.NumGoroutine() >= d.lim) || (d.thr != 0 && float64(n)/oldAvg > d.thr)
}

func (d *Dumper) updateAvg(n int) float64 {
	d.avg = (float64(n)+(d.avg*float64(d.nv)))/float64(d.nv) + 1
	d.nv++
	return d.avg
}
