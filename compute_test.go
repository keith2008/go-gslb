// Our example hello world

package main

import (
	//	"fmt"
	"fmt"
	"strings"
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
	zoneRef := GlobalZoneData()    // Get the latest reference to the zone data
	notrace := NewLookupTraceOff() // Needed for Lookup*

	for _, tt := range tableLookupBackEnd {
		// tt.qname tt.qtype tt.view tt.out
		s := LookupBackEnd(tt.qname, tt.view, false, zoneRef, 0, notrace)

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
	{"example.com", "A", "default", `{[example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},
	{"example.com", "NS", "default", `{[example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"example.com", "SOA", "default", `{[example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"example.com", "TXT", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	{"a.example.com", "A", "default", `{[a.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"a.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},
	{"a.example.com", "NS", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	{"aaaa.example.com", "A", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},
	{"aaaa.example.com", "AAAA", "default", `{[aaaa.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},

	{"ds.example.com", "A", "default", `{[ds.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"ds.example.com", "AAAA", "default", `{[ds.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},

	{"one.example.com", "A", "default", `{[one.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"one.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	{"two.example.com", "A", "default", `{[two.example.com. 300 A 192.0.2.2] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"two.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	{"three.example.com", "A", "default", `{[three.example.com. 300 A 192.0.2.3] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"three.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	{"ds.example.com", "A", "default", `{[ds.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"ds.example.com", "AAAA", "default", `{[ds.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},

	{"expand.example.com", "A", "default", `{[expand.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"expand.example.com", "AAAA", "default", `{[expand.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},

	// Make sure that wildcards do the right thing, as long
	// as they are no more than one hop away from a parent
	// we have SOA for
	{"foo.wildcard.example.com", "A", "default", `{[foo.wildcard.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"foo.wildcard.example.com", "AAAA", "default", `{[foo.wildcard.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"foo.wildcard.example.com", "NS", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},
	{"foo.wildcard.example.com", "SOA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	// Check HC healthchecks, FB fallbacks, and what happens
	// when all HC fail
	{"hc.example.com", "A", "default", `{[hc.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"hc.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	// Try fallback, if the HC nodes are down use FB instead
	{"fb.example.com", "A", "default", `{[fb.example.com. 300 A 192.0.2.3] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"fb.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	// No FB?  Any time HC is specified, and all are down, return all of them instead of empty results.
	// Chances are something is wrong with the monitoring.
	{"nofb.example.com", "A", "default", `{[nofb.example.com. 300 A 192.0.2.1 nofb.example.com. 300 A 192.0.2.2] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"nofb.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	// Local CNAMEs should expand out to IPs.
	{"localcname.example.com", "A", "default", `{[localcname.example.com. 300 A 192.0.2.1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},
	{"localcname.example.com", "AAAA", "default", `{[localcname.example.com. 300 AAAA 2001:db8::1] [example.com. 300 NS ns1.example.com. example.com. 300 NS ns1.example.org.] [ns1.example.com. 300 A 192.0.2.254 ns1.example.com. 300 AAAA 2001:db8::254] true 0}`},

	// As a known side effect: Asking for CNAME on something we can expand, won't give you the CNAME.
	// It'll give the A/AAAA (etc) instead.
	{"localcname.example.com", "CNAME", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 0}`},

	// Foreign CNAMEs should not be expanded, but given to the caller to figure out.
	{"foreigncname.example.com", "A", "default", `{[foreigncname.example.com. 300 CNAME ds.example.org.] [] [] true 0}`},
	{"foreigncname.example.com", "AAAA", "default", `{[foreigncname.example.com. 300 CNAME ds.example.org.] [] [] true 0}`},
	{"foreigncname.example.com", "CNAME", "default", `{[foreigncname.example.com. 300 CNAME ds.example.org.] [] [] true 0}`},

	// Names that don't exist, but under a known SOA
	// Give back 0 answers.. with authority.
	{"dne.example.com", "A", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 3}`},
	{"dne.example.com", "AAAA", "default", `{[] [example.com. 300 SOA ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 86400] [] true 3}`},

	// Not our domain? Should be retreated as non-auth.
	{"dne.example.org", "A", "default", `{[] [] [] false 5}`},
	{"dne.example.org", "AAAA", "default", `{[] [] [] false 5}`},
}

func TestLookupFrontEnd(t *testing.T) {
	initGlobal("t/etc")
	ClearCaches("unit testing TestLookupFrontEnd")
	notrace := NewLookupTraceOff() // Needed for Lookup*

	// NOTE: if we do not clear the caches, the caches are allowed
	// to rotate RRs on us, making the easy string compares not so easy.
	// By purging the cache, we get everyone to a known starting point.

	for _, tt := range tableLookupFrontEnd {
		// tt.qname tt.qtype tt.view tt.out
		s := LookupFrontEndNoCache(tt.qname, tt.view, tt.qtype, 0, notrace)

		found := fmt.Sprintf("%v", s)

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
	notrace := NewLookupTraceOff() // Needed for Lookup*
	b.ReportAllocs()
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = LookupFrontEnd("ipv4.master.test-ipv6.com", "comcast", "A", 0, notrace)

	}
}

func BenchmarkLookupLong(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")
	notrace := NewLookupTraceOff() // Needed for Lookup*
	b.ReportAllocs()
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = LookupFrontEnd("test-ipv6.com", "comcast", "A", 0, notrace)

	}
}

func BenchmarkLookupNoCacheShort(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")
	notrace := NewLookupTraceOff() // Needed for Lookup*
	b.ReportAllocs()
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = LookupFrontEndNoCache("ipv4.master.test-ipv6.com", "comcast", "A", 0, notrace)

	}
}

func BenchmarkLookupNoCacheLong(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")
	notrace := NewLookupTraceOff() // Needed for Lookup*
	b.ReportAllocs()
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = LookupFrontEndNoCache("test-ipv6.com", "comcast", "A", 0, notrace)

	}
}

func BenchmarkWildard1(b *testing.B) {
	initGlobal("t/etc")
	qname := "something.example.org"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		if len(qname) > 2 && qname[0:2] != "*." { // What about the wildcard?
			sp := strings.SplitN(qname, ".", 2) // Split the name into the first hostname, and the remainder
			if len(sp) > 1 {
				try := "*." + sp[1] // Replace the hostname with a *, only if we found a "."
				if try != "*.wildcard.com" {
					b.Fatal("bad test")
				}
			} else {
				b.Fatal("bad test")
			}
		}
	}
}

func BenchmarkWildard2(b *testing.B) {
	initGlobal("t/etc")
	qname := "something.example.org"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		if len(qname) > 2 && qname[0:2] != "*." { // What about the wildcard?
			dot := strings.IndexByte(qname, '.')
			if dot > -1 && dot < len(qname) {
				try := "*" + qname[dot:] // Replace the hostname with a *, only if we found a "."
				if try != "*.wildcard.com" {
					b.Fatalf("bad test 1 dot=%v try=%s", dot, try)
				}
			} else {
				b.Fatal("bad test 2")
			}
		}
	}
}
