//AUTO GENERATED CODE
//\\//go:generated -command doit command ./mygen.pl cache_template.go
//\\//go:generated doit templated_cache_string_string.go            KEYTYPE=string  VALTYPE=string VALNAME=string VALDEFAULT=""
//\\//go:generated doit templated_cache_string_strings.go           KEYTYPE=string  VALTYPE=[]string VALNAME=strings VALDEFAULT=nil
//\\//go:generated doit templated_cache_LookupBEKey_strings.go      KEYTYPE=LookupBEKey VALTYPE=[]string VALNAME=strings  VALDEFAULT=[]string{}
//\\//go:generated doit templated_cache_QueryInfo_LookupResults.go  KEYTYPE=QueryInfo VALTYPE=LookupResults VALNAME=LookupResults  VALDEFAULT=LookupResults{}
//\\//go:generated doit templated_cache_string_dnsRR.go             KEYTYPE=string VALTYPE=dns.RR VALNAME=dnsRR  VALDEFAULT=nil
//\\//go:generated doit templated_cache_QueryInfo_MsgCacheRecord.go KEYTYPE=QueryInfo VALTYPE=MsgCacheRecord VALNAME=MsgCacheRecord VALDEFAULT=MsgCacheRecord{}
//\\//go:generated doit templated_cache_QueryInfo_MsgCacheRecords.go KEYTYPE=QueryInfo VALTYPE=[]MsgCacheRecord VALNAME=MsgCacheRecords VALDEFAULT=[]MsgCacheRecord{}

/*
 * Input template: cache_template.go
 * Output templates: templated_*.go
 * To update those files from this template, use "go generate cache_template.go"
 */

package main

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

func init() {
	_ = dns.TypeA // So that "github.com/miekg/dns" won't disappear
}

// CacheValueContainer_string_dnsRR is a wrapper for dnsRR cached data
type CacheValueContainer_string_dnsRR struct {
	val    dns.RR
	recent bool
}

// CacheContainer_string_dnsRR is a wrapper for a cache keyed on KEY
type CacheContainer_string_dnsRR struct {
	Lock          sync.RWMutex
	Cache         map[string]*CacheValueContainer_string_dnsRR
	SweepInterval time.Duration
}

// NewCache_string_dnsRR creates a new cache, with a cache sweep
// performed at SweepInterval to look for unused entries.  Any
// cache entry not used since the last sweep, is purged.
func NewCache_string_dnsRR(SweepInterval time.Duration) *CacheContainer_string_dnsRR {
	c := new(CacheContainer_string_dnsRR)
	c.SweepInterval = SweepInterval
	c.ClearCache()     // Initializes
	go c.maintenance() // Starts a background threat to keep it tidy
	return c
}

// Get (k string) checks the cache
// for an existing element. returns ok=true on success, with the cached value.
func (c *CacheContainer_string_dnsRR) Get(k string) (v dns.RR, ok bool) {
	c.Lock.Lock() // Read+Write Lock
	vc, ok := c.Cache[k]
	if ok {
		vc.recent = true // We want to keep this entry for at least one pass
	}
	c.Lock.Unlock()
	if ok {
		return vc.val, ok
	}
	return nil, false
}

// Set (k string, v dns.RR) stores
// a value into the cache, and marks the value as "recent" so it survives
// at least one round of cache maintenance.
func (c *CacheContainer_string_dnsRR) Set(k string, v dns.RR) {
	c.Lock.Lock() // Read+Write Lock
	n := new(CacheValueContainer_string_dnsRR)
	n.val = v
	n.recent = true
	c.Cache[k] = n
	c.Lock.Unlock()
}

// ClearCache  immediately and quickly resets
// the entire cache, releasing the old data.  Go's garbage collection
// will clean it when no other references to keys or values exist.
func (c *CacheContainer_string_dnsRR) ClearCache() {
	c.Lock.Lock() // Read+Write Lock
	c.Cache = make(map[string]*CacheValueContainer_string_dnsRR, 16384)
	c.Lock.Unlock()
}

// CleanCache  spends extra time
// examining the cache, looking to see what is "recent".  Anything
// not used recently will be purged.
func (c *CacheContainer_string_dnsRR) CleanCache() {
	// Scan all the values.  Find the values that are not "Recent".  Purge.
	c.Lock.Lock() // Read+Write Lock
	for k, v := range c.Cache {
		if v.recent == false {
			delete(c.Cache, k)
		} else {
			c.Cache[k].recent = false // Mark for next time
		}
	}
	c.Lock.Unlock()
}

// maintenance() is a background thread that will
// call CleanCache periodically. The SweepInterval is passed on creating the cache;
// or can be set by overriding c.SweepInterval.  Changes made to the sweep interval
// will be honored on the following sleep; not during the current sleep.
// One maintenance() goroutine is started per cache instance.
func (c *CacheContainer_string_dnsRR) maintenance() {
	for {
		SleepWithVariance(c.SweepInterval)
		start := time.Now()
		startSize := len(c.Cache)
		c.CleanCache()
		since := time.Since(start)
		stopSize := len(c.Cache)
		Debugf("Cleaned cache string dnsRR time=%s keys: %v -> %v\n", since.String(), startSize, stopSize)

	}
}
