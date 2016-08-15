package ppdump

import (
	"runtime/pprof"
	"sync"
	"time"
)

const (
	_defaultInterval = time.Second
	_defaultThrottle = time.Minute
)

var std *Dumper

// Config contains information required for creating a new Dumper.
type Config struct {
	Profiles     map[string]ProfileOpts // map of profile names to properties
	PollInterval time.Duration          // how often to poll for goroutines. defaults to one second
	Throttle     time.Duration          // time to wait before writing another dump. defaults to one minute
}

// An ActionFunc is a function that will be called when a profile's threshold
// has been exceeded.
type ActionFunc func(profile *pprof.Profile)

// ProfileOpts establishes behavior of when to trigger a dump and what to do
// when that occurs.
type ProfileOpts struct {
	Action    ActionFunc
	Threshold int
}

// Start begins the profiling and dumping procedure using the top level dumper.
func Start(c Config) {
	std = New(c)
	std.Start()
}

// Stop stops the profiling and dumping procedure
func Stop() {
	if std != nil {
		std.Stop()
	}
}

// A Dumper dumps.
// TODO actually describe what it does
type Dumper struct {
	sync.Mutex

	interval  time.Duration
	throttle  time.Duration
	lim       int
	thr       float64
	avg       float64
	nv        int64
	profiles  map[string]ProfileOpts
	stop      chan struct{}
	lastDumps map[string]time.Time
}

// New returns a new Dumper
func New(c Config) *Dumper {
	if c.PollInterval == 0 {
		c.PollInterval = _defaultInterval
	}
	if c.Throttle == 0 {
		c.Throttle = _defaultThrottle
	}
	return &Dumper{
		interval:  c.PollInterval,
		throttle:  c.Throttle,
		profiles:  c.Profiles,
		lastDumps: make(map[string]time.Time),
	}
}

// Start starts the routine of dumping.
func (d *Dumper) Start() {
	d.Lock()
	defer d.Unlock()

	d.stop = make(chan struct{})

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

func (d *Dumper) dump(p ProfileOpts, pp *pprof.Profile) {
	d.Lock()
	defer d.Unlock()

	now := time.Now()
	lastDump := d.lastDumps[pp.Name()]
	if now.Before(lastDump.Add(d.throttle)) {
		return
	}
	d.lastDumps[pp.Name()] = now
	if p.Action != nil {
		p.Action(pp)
	}
}

// Stop stops the runloop. No-op if called more than once.
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
