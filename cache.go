package main

import (
	//	"github.com/davecgh/go-spew/spew"
	"log"
	"time"
)

/*
 *  SEE ALSO:  generator.go, templated_*.go files
 */

// CacheDefaultSweep is the default interval we use for sweeping up the cache
var CacheDefaultSweep = time.Duration(5) * time.Second

// CacheLookupBE implements the "back end" search cache
var CacheLookupBE = NewCache_LookupBEKey_strings("backend", 10000, CacheDefaultSweep)

// CacheLookupFE implmenets the "Front End" (after NS glue, with rcode) cache
// This may be removed later if we decide the dns.Pack byte cache is adequate
var CacheLookupFE = NewCache_QueryInfo_LookupResults("frontend", 10000, CacheDefaultSweep)

// CacheQW is a cache of parsed string -> words
var CacheQW = NewCache_string_strings("quoting", 10000, CacheDefaultSweep)

// CacheView is a cache of IP address -> view name
var CacheView = NewCache_string_string("views", 10000, CacheDefaultSweep)

// CacheRR is a cache of RR strings -> compiled dns.RR objects
var CacheRR = NewCache_string_dnsRR("dnsrr", 10000, CacheDefaultSweep)

// CacheMsgs is a cache of DNS responses previously made to clients
var CacheMsgs = NewCache_QueryInfo_MsgCacheRecords("dnsmsg", 10000, CacheDefaultSweep)

// LookupBEKey is a map key for getting expanded strings from zone data
type LookupBEKey struct {
	qname  string
	view   string
	skipHC bool
}

// QueryInfo defines the common ways we segregate the cache.
// The qname is obvious; but we also take into account the
// query type ("A","AAAA", etc) as well as what view the
// caller is from ("comcast","default",etc).
type QueryInfo struct {
	qname string
	view  string
	qtype string
}

// MsgCacheRecord Contains the packed binary response, and the rcode for statistics purposes
type MsgCacheRecord struct {
	msg      []byte
	rcodeStr string
}

// ClearCaches will reset all query related caches to empty.
// Old instances may still persist until no longer referenced
// and then cleaned up by GC.
func ClearCaches(reason string) {

	log.Printf("Clearing all caches: %s\n", reason)
	CacheLookupBE.ClearCache()
	CacheLookupFE.ClearCache()
	CacheQW.ClearCache()
	CacheView.ClearCache()
	CacheRR.ClearCache()
}

// Satisfy generator.go during editing.
// generator.go is used as a template for caches.

// KEYTYPE is our cache key
type KEYTYPE string

// VALTYPE is our cahe value
type VALTYPE string

// VALDEFAULT is our default value of the key does not exist
var VALDEFAULT = ""
