package main

import (
	"fmt"
	"testing"

	"github.com/miekg/dns"
)

var tableFindView = []struct {
	in  string
	out string
}{
	{"50.184.213.245:12345", "view=comcast asn=7922 isp=Comcast Cable country=US"},
	{"[2601:647:4900:78ae:d497:ef6b:9e49:d98]:12345", "view=comcast asn=7922 isp=Comcast Cable country=US"},
	{"[206.190.36.45]:12345", "view=default asn=36647 isp=Yahoo! Broadcast Services country=US"},
	{"[216.218.228.114]:12345", "view=default asn=6939 isp=Hurricane Electric country=US"},
}

func TestFindView(t *testing.T) {
	initGlobal("t/etc")

	for _, tt := range tableFindView {
		// tt.in tt.out
		view, asnString, ispString, country := findView(tt.in)
		found := fmt.Sprintf("view=%v asn=%v isp=%v country=%v", view, asnString, ispString, country)

		if found == tt.out {
			t.Logf("findView(%v) good", tt.in)
		} else {
			t.Errorf("findView(%v) should return %v, found %v", tt.in, tt.out, found)
		}
	}
}

func TestDNSNewRR(t *testing.T) {
	rr, err := dns.NewRR("example.com. 3600 IN A 10.2.3.4")
	t.Logf("rr=%v err=%v", rr, err)

	// Go with the "blessed results" approach.
	want :=
		`&dns.A{Hdr:dns.RR_Header{Name:"example.com.", Rrtype:0x1, Class:0x1, Ttl:0xe10, Rdlength:0x0}, A:net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xa, 0x2, 0x3, 0x4}}`
	have := fmt.Sprintf("%#v", rr)
	if want != have {
		t.Logf("wanted: %s", want)
		t.Fatalf("found: %s", have)
	}

}

func BenchmarkDNSNewRR(b *testing.B) {
	// Expensive stuff first
	b.ReportAllocs()
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_, _ = dns.NewRR("example.com. 3600 IN A 10.2.3.4")

	}
}

func BenchmarkDNSCopyRR(b *testing.B) {
	// Expensive stuff first
	rr, _ := dns.NewRR("example.com. 3600 IN A 10.2.3.4")
	b.ReportAllocs()
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = dns.Copy(rr)
	}
}

func BenchmarkFindViewIpNoPort(b *testing.B) {
	// Expensive stuff first
	ip := "50.184.213.245"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, _, _ = findView(ip)
	}
}

func BenchmarkFindViewIpPort(b *testing.B) {
	ip := "50.184.213.245:12345"
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, _, _ = findView(ip)
	}
}

func BenchmarkFindViewCached(b *testing.B) {
	ip := "50.184.213.245:12345"
	_, asn, _, _ := findView(ip)
	CacheView.Set(ip, asn)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = CacheView.Get(ip)
	}
}
