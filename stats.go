package main

import (
	"expvar"
	"strconv"
	"time"
)

type statsBundleType struct {
	name     string
	counters *expvar.Map
	snapshot map[string]int64 // not advertised
	qpm      *expvar.Map
}

var statsQuery = newStat("query")
var statsResponse = newStat("response")
var statsMaxMind = newStat("maxmind")
var statsCache = newStat("cache")

func (b *statsBundleType) Increment(s string) {
	b.counters.Add(s, 1)
}

// This will be called as a goroutine.
// This will every 60 seconds, walk the b.counters hash;
// compute deltas (and rates) for the qpm hash; and then
// save the current values for the next round.
func (b *statsBundleType) periodic() {

	// For the current KeyValue, compute the rates
	// and save the last known values.
	doHelper := func(v expvar.KeyValue) {
		key := v.Key
		valueStr := v.Value.String() // Really, I can only get a string?
		value, _ := strconv.ParseInt(valueStr, 10, 64)

		// Can we compute a delta? If so, figure out the query rate
		if previous, found := b.snapshot[key]; found {
			delta := value - previous // How many from last time until now?
			qpmVar := new(expvar.Int) // expvar's "set" interface *demands* a expvar.Int
			qpmVar.Set(delta)         // .. which then needs the value stored afterwords
			b.qpm.Set(key, qpmVar)    // And then we can finally set this gauge value.
		}

		// Save into the snapshot, for next time around
		b.snapshot[key] = value
	}

	for {
		b.counters.Do(doHelper)
		time.Sleep(time.Duration(60) * time.Second)
	}
}

// Starts a new set of counters and gauges.
// Spins off a background thread to update gauges.
func newStat(name string) *statsBundleType {
	b := new(statsBundleType)
	b.name = name
	b.counters = expvar.NewMap(name + "_counter")
	b.qpm = expvar.NewMap(name + "_qpm")
	b.snapshot = make(map[string]int64)

	go b.periodic()

	return b
}
