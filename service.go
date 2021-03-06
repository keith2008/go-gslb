package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
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
	if exists == false {
		HealthChecks.Status[ServiceTargetKey{service, target}] = false
		go backgroundServiceCheck(service, target, secs)
	}
	HealthChecks.Lock.Unlock() //RW
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
		SleepWithVariance(t)

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

func dumpHealthCheckStatusAsText() string {
	ret := make([]string, 0, 0)
	// Copy the status, with as minimal time as possible inside the lock
	HealthChecks.Lock.Lock() // RW
	for key, val := range HealthChecks.Status {
		s := fmt.Sprintf("%s %s: %v\n", key.Service, key.Target, val)
		ret = append(ret, s)
	}
	HealthChecks.Lock.Unlock() // RW
	sort.Strings(ret)
	retStr := strings.Join(ret, "")
	return retStr
}

func myHTTPHealthHandler(w http.ResponseWriter, r *http.Request) {
	s := dumpHealthCheckStatusAsText()
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, s)
}

func init() {
	http.HandleFunc("/gslb/hc", myHTTPHealthHandler)
	http.HandleFunc("/gslb/healthcheck", myHTTPHealthHandler)

}
