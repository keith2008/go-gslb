//AUTO GENERATED CODE
//\\//go:generated -command doit  ./mygen.pl cache_template.go
//\\//go:generated doit templated_cache_string_string.go            KEYTYPE=string  VALTYPE=string VALNAME=string VALDEFAULT=""
//\\//go:generated doit templated_cache_string_strings.go           KEYTYPE=string  VALTYPE=[]string VALNAME=strings VALDEFAULT=nil
//\\//go:generated doit templated_cache_LookupBEKey_strings.go      KEYTYPE=LookupBEKey VALTYPE=[]string VALNAME=strings  VALDEFAULT=[]string{}
//\\//go:generated doit templated_cache_QueryInfo_LookupResults.go  KEYTYPE=QueryInfo VALTYPE=LookupResults VALNAME=LookupResults  VALDEFAULT=LookupResults{}
//\\//go:generated doit templated_cache_string_dnsRR.go             KEYTYPE=string VALTYPE=dns.RR VALNAME=dnsRR  VALDEFAULT=nil
//\\//go:generated doit templated_cache_QueryInfo_MsgCacheRecords.go KEYTYPE=QueryInfo VALTYPE=[]MsgCacheRecord VALNAME=MsgCacheRecords VALDEFAULT=[]MsgCacheRecord{}

/*
 * Input template: cache_template.go
 * Output templates: templated_*.go
 * To update those files from this template, use "go generate cache_template.go"
 */

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
)

func init() {
	_ = dns.TypeA // So that "github.com/miekg/dns" won't disappear
}

// CacheValueContainer_LookupBEKey_strings is a wrapper for strings cached data
type CacheValueContainer_LookupBEKey_strings struct {
	val    []string
	recent bool
}

// CacheContainer_LookupBEKey_strings is a wrapper for a cache keyed on KEY
type CacheContainer_LookupBEKey_strings struct {
	Lock          sync.RWMutex
	Cache         map[LookupBEKey]*CacheValueContainer_LookupBEKey_strings
	SweepInterval time.Duration
	StatsName     string
	cHit          string
	cMiss         string
	cSet          string
	cFull         string
	MaxSize       int
}

// NewCache_LookupBEKey_strings creates a new cache, with a cache sweep
// performed at SweepInterval to look for unused entries.  Any
// cache entry not used since the last sweep, is purged.
func NewCache_LookupBEKey_strings(StatsName string, MaxSize int, SweepInterval time.Duration) *CacheContainer_LookupBEKey_strings {
	c := new(CacheContainer_LookupBEKey_strings)
	c.SweepInterval = SweepInterval
	c.StatsName = StatsName
	c.MaxSize = MaxSize
	c.cHit = fmt.Sprintf("%s:hit", c.StatsName)
	c.cMiss = fmt.Sprintf("%s:miss", c.StatsName)
	c.cSet = fmt.Sprintf("%s:set", c.StatsName)
	c.cFull = fmt.Sprintf("%s:full", c.StatsName)
	c.ClearCache()     // Initializes
	go c.maintenance() // Starts a background threat to keep it tidy
	return c
}

// Get (k LookupBEKey) checks the cache
// for an existing element. returns ok=true on success, with the cached value.
func (c *CacheContainer_LookupBEKey_strings) Get(k LookupBEKey) (v []string, ok bool) {
	c.Lock.Lock() // Read+Write Lock
	vc, ok := c.Cache[k]
	if ok {
		vc.recent = true // We want to keep this entry for at least one pass
	}
	c.Lock.Unlock()
	if ok {
		statsCache.Increment(c.cHit)
		return vc.val, ok
	}
	statsCache.Increment(c.cMiss)
	return []string{}, false
}

// Set (k LookupBEKey, v []string) stores
// a value into the cache, and marks the value as "recent" so it survives
// at least one round of cache maintenance.
func (c *CacheContainer_LookupBEKey_strings) Set(k LookupBEKey, v []string) {
	saved := false
	c.Lock.Lock() // Read+Write Lock
	if len(c.Cache) < c.MaxSize {
		n := new(CacheValueContainer_LookupBEKey_strings)
		n.val = v
		n.recent = true
		c.Cache[k] = n
		saved = true
	}
	c.Lock.Unlock()
	if saved {
		statsCache.Increment(c.cSet)
	} else {
		statsCache.Increment(c.cFull)
	}
}

func (c *CacheContainer_LookupBEKey_strings) CheckConfig() {
	if GlobalConfigAvailable() {
		config := GlobalConfig()
		c.Lock.Lock() // Read+Write Lock
		name := c.StatsName
		c.Lock.Unlock()

		if val, ok := config.GetSectionNameValueInt("interval", "clean_cache"); ok {
			c.SetInterval(time.Duration(val) * time.Second)
		}
		if val, ok := config.GetSectionNameValueInt("cachesize", name); ok {
			c.SetMaxSize(val)
		}
	}
}

// ClearCache  immediately and quickly resets
// the entire cache, releasing the old data.  Go's garbage collection
// will clean it when no other references to keys or values exist.
func (c *CacheContainer_LookupBEKey_strings) ClearCache() {
	c.CheckConfig() // Get latest values for sleeping, max size
	c.Lock.Lock()   // Read+Write Lock
	c.Cache = make(map[LookupBEKey]*CacheValueContainer_LookupBEKey_strings, 16384)
	c.Lock.Unlock()
}

// CleanCache  spends extra time
// examining the cache, looking to see what is "recent".  Anything
// not used recently will be purged.
func (c *CacheContainer_LookupBEKey_strings) CleanCache() {
	// Scan all the values.  Find the values that are not "Recent".  Purge.
	c.Lock.Lock() // Read+Write Lock
	start := time.Now()
	startSize := len(c.Cache)
	for k, v := range c.Cache {
		if v.recent == false {
			delete(c.Cache, k)
		} else {
			c.Cache[k].recent = false // Mark for next time
		}
	}
	since := time.Since(start)
	stopSize := len(c.Cache)
	c.Lock.Unlock()
	Debugf("Cleaned cache LookupBEKey strings time=%s keys: %v -> %v\n", since.String(), startSize, stopSize)

}

// maintenance() is a background thread that will
// call CleanCache periodically. The SweepInterval is passed on creating the cache;
// or can be set by overriding c.SweepInterval.  Changes made to the sweep interval
// will be honored on the following sleep; not during the current sleep.
// One maintenance() goroutine is started per cache instance.
func (c *CacheContainer_LookupBEKey_strings) maintenance() {
	for {
		SleepWithVariance(c.SweepInterval)
		c.CleanCache()
	}
}

func (c *CacheContainer_LookupBEKey_strings) SetMaxSize(MaxSize int) {
	c.Lock.Lock() // Read+Write Lock
	c.MaxSize = MaxSize
	c.Lock.Unlock()
}
func (c *CacheContainer_LookupBEKey_strings) GetMaxSize() (MaxSize int) {
	c.Lock.Lock() // Read+Write Lock
	MaxSize = c.MaxSize
	c.Lock.Unlock()
	return MaxSize
}

func (c *CacheContainer_LookupBEKey_strings) SetInterval(t time.Duration) {
	c.Lock.Lock() // Read+Write Lock
	c.SweepInterval = t
	c.Lock.Unlock()
}
func (c *CacheContainer_LookupBEKey_strings) GetInterval() time.Duration {
	c.Lock.Lock() // Read+Write Lock
	t := c.SweepInterval
	c.Lock.Unlock()
	return t
}
