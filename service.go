package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

// ServiceTargetKey is a map key for the global service checks status
type ServiceTargetKey struct {
	Service string
	Target  string
}

// Checks is the structure that holds the global service checks plus a mutex for accessing
type Checks struct {
	Lock   sync.RWMutex
	Status map[ServiceTargetKey]bool
}

// HealthChecks contains the current status of all backgrounded health checks.
// Use the accessor functions to read/write; this must be kept thread safe.
var HealthChecks Checks

func init() {
	HealthChecks.Status = make(map[ServiceTargetKey]bool)
}

// AddCheck starts a particular service check, against a specific target; with checks every "time" (give or take a random amount)
// Once started, the checks won't stop (short of an application reset).
// At startup, the service check will be DOWN, and require at least one poll of the service to go to UP.
// TODO provide a hook to clear/rescan and populate service checks.
func AddCheck(service string, target string, secs int) (exists bool) {

	exists = false
	HealthChecks.Lock.Lock() // RW
	if _, ok := HealthChecks.Status[ServiceTargetKey{service, target}]; ok {
		exists = true
	}
	HealthChecks.Lock.Unlock() //RW
	if exists == false {
		HealthChecks.Status[ServiceTargetKey{service, target}] = false
		go backgroundServiceCheck(service, target, secs)
	}
	return exists
}

// backgroundServiceCheck will start monitoring a given service for a specifieid target.
// The interval must be specified (in go's time.Time format).
func backgroundServiceCheck(service string, target string, secs int) {
	t := time.Duration(secs) * time.Second
	for {
		// Get the latest debug flag.
		status, err := dispatchServiceCheck(service, target)
		changed, ok := SetStatus(service, target, status)

		// Make some noise about it.
		if ok {
			if changed {
				ClearCaches("health check status changed")
				log.Printf("service %s target %s status %v changed %v err %v\n", service, target, status, changed, err)

			}
		} else {
			Debugf("Lost our place! service %s target %s status %v changed %v err %v\n", service, target, status, changed, err)
			return // Exit goroutine, we have no more work.
		}

		// Sleep some - vary the amounts a bit.
		amt := 0.9 + rand.Float64()/5.0       // 0.9x to 1.1x
		t2 := time.Duration(float64(t) * amt) // .. of the original amount
		time.Sleep(t2)

	}
}

// GetStatus gets the status of a service for a given target.
// Use only if "ok", otherwise assume that the status is not (yet?) recorded.
func GetStatus(service string, target string) (status bool, ok bool) {
	HealthChecks.Lock.RLock()                                           // RO
	check, ok := HealthChecks.Status[ServiceTargetKey{service, target}] // Get old status
	HealthChecks.Lock.RUnlock()                                         // RO
	return check, ok                                                    // Let the caller know the old status
}

// SetStatus puts the status of a service for a given target.
// Returns "changed", indicating if the new value is different from the old value.
// Use only if "ok".
func SetStatus(service string, target string, status bool) (changed bool, ok bool) {
	HealthChecks.Lock.Lock()                                          // RW
	old, ok := HealthChecks.Status[ServiceTargetKey{service, target}] // Get old status
	HealthChecks.Status[ServiceTargetKey{service, target}] = status   // Set status
	HealthChecks.Lock.Unlock()                                        // RW
	return old != status, ok                                          // Let the caller know if things "changed"
}

func empty(service string, target string, status bool) {
	return
}
