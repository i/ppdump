package ppdump

import (
	"bytes"
	"fmt"
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
	Interval     time.Duration      `validate:"nonzero"` // how often to poll for goroutines
	Profiles     map[string]Profile // list of profiles to track
	Writer       Writer             // where to write the dump to
	Throttle     time.Duration      // time to wait before writing another dump
	ThresholdPct float64            // percentage of increase to trigger a dump
}

type Profile struct {
	Threshold int
	Debug     int
}

type Writer interface {
	Write(profile string, threshold int, dump []byte)
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

	interval  time.Duration
	throttle  time.Duration
	lim       int
	thr       float64
	avg       float64
	nv        int64
	writer    Writer
	profiles  map[string]Profile
	stop      chan struct{}
	lastDumps map[string]time.Time
}

func NewDumper(c Config) (*Dumper, error) {
	if c.Writer == nil {
		return nil, fmt.Errorf("no writer")
	}
	if err := validator.Validate(c); err != nil {
		return nil, err
	}

	return &Dumper{
		interval:  c.Interval,
		throttle:  c.Throttle,
		thr:       c.ThresholdPct,
		profiles:  c.Profiles,
		writer:    c.Writer,
		lastDumps: make(map[string]time.Time),
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
			d.checkAndDump()
		case <-d.stop:
			return
		}
	}
}

func (d *Dumper) dump(p Profile, pp *pprof.Profile) {
	d.Lock()
	defer d.Unlock()

	now := time.Now()
	lastDump := d.lastDumps[pp.Name()]
	if now.Before(lastDump.Add(d.throttle)) {
		return
	}
	d.lastDumps[pp.Name()] = now

	buf := bytes.NewBuffer(nil)
	pp.WriteTo(buf, p.Debug)
	d.writer.Write(pp.Name(), p.Threshold, buf.Bytes())
}

// Stops the runloop. No-op if called more than once.
func (d *Dumper) Stop() {
	d.Lock()
	defer d.Unlock()
	defer func() { recover() }() // doesn't matter if the channel is closed twice
	close(d.stop)
}

func (d *Dumper) checkAndDump() {
	for name, p := range d.profiles {
		if profile := pprof.Lookup(name); profile != nil {
			if profile.Count() > p.Threshold {
				d.dump(p, profile)
			}
		}
	}
}
