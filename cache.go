package main

import (
	//	"github.com/davecgh/go-spew/spew"
	"log"
	"sync"

	"github.com/miekg/dns"
)

// LookupBEKey is a map key for getting expanded strings from zone data
type LookupBEKey struct {
	askedName string
	view      string
	skipHC    bool
}

// LookupBECacheType is a cache for holding expanded strings from zone data
type LookupBECacheType struct {
	Lock  sync.RWMutex
	Cache map[LookupBEKey][]string
}

// LookupFEKey is a map key for LookupFECache
type LookupFEKey struct {
	askedName string
	view      string
	qtype     string
}

//LookupFECacheType holds DNS-ready LookupResults
type LookupFECacheType struct {
	Lock    sync.RWMutex
	Cache   map[LookupFEKey]LookupResults
	MaxSize int
}

// LookupQWCacheType is a cache for holding tokenized strings
type LookupQWCacheType struct {
	Lock  sync.RWMutex
	Cache map[string][]string
}

// LookupQWCacheType is a cache for holding tokenized strings
type LookupViewCacheType struct {
	Lock    sync.RWMutex
	Cache   map[string]string
	MaxSize int
}

// RRCacheType is a cache for holding DNS resource records parsed from strings
type RRCacheType struct {
	Lock  sync.RWMutex
	Cache map[string]dns.RR
}

// LookupBECache is the "backend" cache for generic "Get me all records for this name and view"
var LookupBECache LookupBECacheType

// LookupFECache is the "frontend" cache with DNS-ready LookupResults, including glue
var LookupFECache LookupFECacheType

// LookupQWCache is the "frontend" cache with DNS-ready LookupResults, including glue
var LookupQWCache LookupQWCacheType

// LookupViewCache caches IP -> zone name
var LookupViewCache LookupViewCacheType

// RRCache caches strings parsed into DNS RR objects
var RRCache RRCacheType

func init() {
	InitCaches("startup", 10000)
	return
}

// InitCaches will brutally stomp the caches right now,
// with zero dependency on configuration reading.
func InitCaches(reason string, size int) {

	log.Printf("InitCaches cache: %s\n", reason)

	// Front End cache
	LookupFECache.Lock.Lock()                                         // RW
	LookupFECache.Cache = make(map[LookupFEKey]LookupResults, size*2) // Initialize a clean map
	LookupFECache.MaxSize = size                                      // Set the upper limit on size until we purge
	LookupFECache.Lock.Unlock()                                       // RW

	// Back End cache
	LookupBECache.Lock.Lock()                                    // RW
	LookupBECache.Cache = make(map[LookupBEKey][]string, size*2) // Initialize a clean map
	LookupBECache.Lock.Unlock()                                  // RW

	// Quoted Words parser cache
	LookupQWCache.Lock.Lock()                               // RW
	LookupQWCache.Cache = make(map[string][]string, size*2) // Initialize a clean map
	LookupQWCache.Lock.Unlock()                             // RW

	// Quoted Words parser cache
	RRCache.Lock.Lock()                             // RW
	RRCache.Cache = make(map[string]dns.RR, size*2) // Initialize a clean map
	RRCache.Lock.Unlock()                           // RW

	// IP to zone cache
	LookupViewCache.Lock.Lock()                             // RW
	LookupViewCache.Cache = make(map[string]string, size*2) // Initialize a clean map
	LookupViewCache.Lock.Unlock()                           // RW

	return
}

// ClearCaches will reset all query related caches to empty.
// Old instances may still persist until no longer referenced
// and then cleaned up by GC.
func ClearCaches(reason string) {
	// Grab the latest configuration for maxsize
	// before we lock the cache.  Keep the work inside the Lock
	// to an absolute minimum.
	i, ok := GlobalConfig().GetSectionNameValueInt("default", "maxcache")
	if !ok {
		i = 10000
	}
	InitCaches(reason, i)
	return
}

// setLookupBECache updates the "back end" cache with matching strings.
func setLookupBECache(askedName string, view string, skipHC bool, s []string) {
	key := LookupBEKey{askedName, view, skipHC}
	LookupBECache.Lock.Lock() // RW
	LookupBECache.Cache[key] = s
	LookupBECache.Lock.Unlock() // RW
	return
}

// getLookupBECache reads from the "back end" cache; only use results if "ok" is true
func getLookupBECache(askedName string, view string, skipHC bool) (s []string, ok bool) {
	key := LookupBEKey{askedName, view, skipHC}
	LookupBECache.Lock.RLock() // RO
	s, ok = LookupBECache.Cache[key]
	LookupBECache.Lock.RUnlock() // RO
	return s, ok
}

// setLookupFECache updates the "front end" cache with prepared LookupResults
func setLookupFECache(askedName string, view string, qtype string, s LookupResults) {
	key := LookupFEKey{askedName, view, qtype}

	// Do as little as possible inside the lock
	LookupFECache.Lock.Lock()                                     // RW
	LookupFECache.Cache[key] = s                                  // Store LookupResults
	needClear := len(LookupFECache.Cache) > LookupFECache.MaxSize // Check cache size while we are locked
	LookupFECache.Lock.Unlock()                                   // RW
	if needClear {                                                // If the cache is huge, flush it.
		ClearCaches("FE cache grew too big") // And ignore the cache write.
	}

	return
}

// getLookupFECache reads from the "front end" cache; only use if "ok" is true
func getLookupFECache(askedName string, view string, qtype string) (s LookupResults, ok bool) {
	key := LookupFEKey{askedName, view, qtype}
	LookupFECache.Lock.RLock() // RW
	s, ok = LookupFECache.Cache[key]

	// Doing this rotation in the cache read is dirty; but it avoids opening the lock a second time.
	// Yes, it is dirty, and yes, this time, this is intentional.
	if ok && len(s.Ans) > 1 { // If we have more than one result
		s.Ans = append(s.Ans[1:], s.Ans[0]) // Rotate while we have this open
		LookupFECache.Cache[key] = s        // And write it back
	}
	LookupFECache.Lock.RUnlock() // RW
	return s, ok
}

// setLookupQWCache updates the "back end" cache with matching strings.
func setLookupQWCache(key string, s []string) {
	LookupQWCache.Lock.Lock() // RW
	LookupQWCache.Cache[key] = s
	LookupQWCache.Lock.Unlock() // RW
	return
}

// getLookupBECache reads from the "back end" cache; only use results if "ok" is true
func getLookupQWCache(key string) (s []string, ok bool) {
	LookupQWCache.Lock.RLock() // RO
	s, ok = LookupQWCache.Cache[key]
	LookupQWCache.Lock.RUnlock() // RO
	return s, ok
}

// setLookupQWCache updates the "back end" cache with matching strings.
func setRRCache(key string, rr dns.RR) {
	RRCache.Lock.Lock() // RW
	RRCache.Cache[key] = rr
	RRCache.Lock.Unlock() // RW
	return
}

// getLookupBECache reads from the "back end" cache; only use results if "ok" is true
func getRRCache(key string) (rr dns.RR, ok bool) {
	RRCache.Lock.RLock() // RO
	rr, ok = RRCache.Cache[key]
	RRCache.Lock.RUnlock() // RO
	return rr, ok
}

// setLookupQWCache updates the "back end" cache with matching strings.
func setLookupViewCache(key string, s string) {
	LookupViewCache.Lock.Lock()                                       // RW
	needClear := len(LookupViewCache.Cache) > LookupViewCache.MaxSize // Check cache size while we are locked
	if needClear {
		log.Printf("Clearing LookupViewCache due to size\n")
		LookupViewCache.Cache = make(map[string]string, LookupViewCache.MaxSize) // Initialize a clean map
	}
	LookupViewCache.Cache[key] = s
	LookupViewCache.Lock.Unlock() // RW
	return
}

// getLookupBECache reads from the "back end" cache; only use results if "ok" is true
func getLookupViewCache(key string) (s string, ok bool) {
	LookupViewCache.Lock.RLock() // RO
	s, ok = LookupViewCache.Cache[key]
	LookupViewCache.Lock.RUnlock() // RO
	return s, ok
}
