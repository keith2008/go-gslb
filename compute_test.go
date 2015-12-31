// Our example hello world

package main

import (
	//	"fmt"
	"fmt"
	"testing"

	//	"time"
)

var tableLookupBackEnd = []struct {
	qname string
	view  string
	out   string
}{
	{"example.com", "default", `[SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400 NS ns1.example.com NS ns1.example.org MX 10 example.com A 192.0.2.1]`},
	{"a.example.com", "default", `[A 192.0.2.1]`},
	{"aaaa.example.com", "default", `[AAAA 2001:db8::1]`},
	{"ds.example.com", "default", `[A 192.0.2.1 AAAA 2001:db8::1]`},
	{"one.example.com", "default", `[A 192.0.2.1]`},
	{"two.example.com", "default", `[A 192.0.2.2]`},
	{"three.example.com", "default", `[A 192.0.2.3]`},
	{"expand.example.com", "default", `[A 192.0.2.1 AAAA 2001:db8::1]`},
	{"foo.wildcard.example.com", "default", `[A 192.0.2.1 AAAA 2001:db8::1]`},
	{"hc.example.com", "default", `[A 192.0.2.1]`},
	{"fb.example.com", "default", `[A 192.0.2.3]`},
	{"nofb.example.com", "default", `[A 192.0.2.1 A 192.0.2.2]`},
	{"localcname.example.com", "default", `[A 192.0.2.1 AAAA 2001:db8::1]`},
	{"foreigncname.example.com", "default", `[CNAME ds.example.org]`},
	{"dne.example.com", "default", `[]`},
}

func TestLookupBackEnd(t *testing.T) {
	initGlobal("t/etc")
	ClearCaches("unit testing TestLookupBackEnd")
	zoneRef := GlobalZoneData() // Get the latest reference to the zone data

	for _, tt := range tableLookupBackEnd {
		// tt.qname tt.qtype tt.view tt.out
		s := LookupBackEnd(tt.qname, tt.view, false, 2, zoneRef, nil)

		found := fmt.Sprintf("%s", s)

		if found == tt.out {
			t.Logf("LookupBackEnd(zoneRef,%v,%v) good", tt.qname, tt.view)
		} else {
			t.Errorf("LookupBackEnd(zoneRef,%v,%v) should return %v, found %v", tt.qname, tt.view, tt.out, found)
		}
	}
}

var tableLookupFrontEnd = []struct {
	qname string
	qtype string
	view  string
	out   string
}{
	// top level
	{"example.com", "A", "default", `{[example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},
	{"example.com", "NS", "default", `{[example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"example.com", "SOA", "default", `{[example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"example.com", "TXT", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	{"a.example.com", "A", "default", `{[a.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"a.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},
	{"a.example.com", "NS", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	{"aaaa.example.com", "A", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},
	{"aaaa.example.com", "AAAA", "default", `{[aaaa.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},

	{"ds.example.com", "A", "default", `{[ds.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"ds.example.com", "AAAA", "default", `{[ds.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},

	{"one.example.com", "A", "default", `{[one.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"one.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	{"two.example.com", "A", "default", `{[two.example.com. 300 A 192.0.2.2] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"two.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	{"three.example.com", "A", "default", `{[three.example.com. 300 A 192.0.2.3] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"three.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	{"ds.example.com", "A", "default", `{[ds.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"ds.example.com", "AAAA", "default", `{[ds.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},

	{"expand.example.com", "A", "default", `{[expand.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"expand.example.com", "AAAA", "default", `{[expand.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},

	// Make sure that wildcards do the right thing, as long
	// as they are no more than one hop away from a parent
	// we have SOA for
	{"foo.wildcard.example.com", "A", "default", `{[foo.wildcard.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"foo.wildcard.example.com", "AAAA", "default", `{[foo.wildcard.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"foo.wildcard.example.com", "NS", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},
	{"foo.wildcard.example.com", "SOA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	// Check HC healthchecks, FB fallbacks, and what happens
	// when all HC fail
	{"hc.example.com", "A", "default", `{[hc.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"hc.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	// Try fallback, if the HC nodes are down use FB instead
	{"fb.example.com", "A", "default", `{[fb.example.com. 300 A 192.0.2.3] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"fb.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	// No FB?  Any time HC is specified, and all are down, return all of them instead of empty results.
	// Chances are something is wrong with the monitoring.
	{"nofb.example.com", "A", "default", `{[nofb.example.com. 300 A 192.0.2.1 nofb.example.com. 300 A 192.0.2.2] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"nofb.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	// Local CNAMEs should expand out to IPs.
	{"localcname.example.com", "A", "default", `{[localcname.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},
	{"localcname.example.com", "AAAA", "default", `{[localcname.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] %!s(bool=true) %!s(int=0)}`},

	// As a known side effect: Asking for CNAME on something we can expand, won't give you the CNAME.
	// It'll give the A/AAAA (etc) instead.
	{"localcname.example.com", "CNAME", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=0)}`},

	// Foreign CNAMEs should not be expanded, but given to the caller to figure out.
	{"foreigncname.example.com", "A", "default", `{[foreigncname.example.com. 300 CNAME ds.example.org.] [] [] %!s(bool=true) %!s(int=0)}`},
	{"foreigncname.example.com", "AAAA", "default", `{[foreigncname.example.com. 300 CNAME ds.example.org.] [] [] %!s(bool=true) %!s(int=0)}`},
	{"foreigncname.example.com", "CNAME", "default", `{[foreigncname.example.com. 300 CNAME ds.example.org.] [] [] %!s(bool=true) %!s(int=0)}`},

	// Names that don't exist, but under a known SOA
	// Give back 0 answers.. with authority.
	{"dne.example.com", "A", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=3)}`},
	{"dne.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] %!s(bool=true) %!s(int=3)}`},

	// Not our domain? Should be retreated as non-auth.
	{"dne.example.org", "A", "default", `{[] [] [] %!s(bool=false) %!s(int=5)}`},
	{"dne.example.org", "AAAA", "default", `{[] [] [] %!s(bool=false) %!s(int=5)}`},
}

func TestLookupFrontEnd(t *testing.T) {
	initGlobal("t/etc")
	ClearCaches("unit testing TestLookupFrontEnd")

	// NOTE: if we do not clear the caches, the caches are allowed
	// to rotate RRs on us, making the easy string compares not so easy.
	// By purging the cache, we get everyone to a known starting point.

	for _, tt := range tableLookupFrontEnd {
		// tt.qname tt.qtype tt.view tt.out
		s := LookupFrontEndNoCache(tt.qname, tt.view, tt.qtype, nil)

		found := fmt.Sprintf("%s", s)

		if found == tt.out {
			t.Logf("LookupFrontEnd(zoneRef,%v,%v,%v) good", tt.qname, tt.view, tt.qtype)
		} else {
			t.Errorf("LookupFrontEnd(zoneRef,%v,%v,%v) should return %v, found %v", tt.qname, tt.view, tt.qtype, tt.out, found)
		}
	}
}

func BenchmarkLookupShort(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = LookupFrontEnd("ipv4.master.test-ipv6.com", "comcast", "A", nil)

	}
}

func BenchmarkLookupLong(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = LookupFrontEnd("test-ipv6.com", "comcast", "A", nil)

	}
}

// Compare a ew things
// QuotedStringToWords(s)
// map[string]..
// simple dumb array

func BenchmarkQWuncached(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")

	s := "SOA ns1.test-ipv6.com. jfesler.test-ipv6.com. 2010050801 10800 3600 604800 86400"
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = QuotedStringToWords(s)

	}
}

func BenchmarkQWcached(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")

	s := ""
	v := []string{}
	for i := 1; i < 1000; i++ {
		s = fmt.Sprintf("SOA ns1.test-ipv6.com. jfesler.test-ipv6.com. 2010050801 10800 3600 604800 %v", i)
		v = QuotedStringToWords(s)
		setLookupQWCache(s, v)
	}

	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		v, _ = getLookupQWCache(s)

	}
}
