// Our example hello world
package main

// https://golang.org/pkg/sort/

import (
	"fmt"
	"testing"
)

var tableTestGeoIP2ASN = []struct {
	in  string
	out string
}{
	{"50.184.213.245", "7922"},
	{"2001:4998::1", "10310"},
}

var tableTestGeoIP2ISP = []struct {
	in  string
	out string
}{
	{"50.184.213.245", "Comcast Cable"},
	{"2001:4998::1", "Yahoo!"},
	{"2620:0:cca::17", "OpenDNS, LLC"},
	{"2620:0:cca::21", "OpenDNS, LLC"},
}

var tableTestGeoIP2Country = []struct {
	in  string
	out string
}{
	{"50.184.213.245", "US"},
	{"2001:4998::1", "US"},
	{"2620:0:cca::17", "SG"},
	{"2600:6:ff82:1:66:1:68:137", "US"},
	{"2620:0:cca::21", "SG"},
}

func TestGeoIP2ASN(t *testing.T) {
	initGlobal("t/etc")

	m, err := NewGeoIP2("/var/lib/GeoIP/GeoIP2-ISP.mmdb")
	if err != nil {
		t.Errorf("Error loading data: %v", err)
		return
	}

	for _, tt := range tableTestGeoIP2ASN {
		// tt.in tt.out
		record, err := m.ISP(tt.in)

		// tt.in tt.out
		if err != nil {
			t.Fatal(err)
		}
		found := fmt.Sprintf("%v", record.AutonomousSystemNumber)
		if found == tt.out {
			t.Logf("Lookup(%v) good", tt.in)
		} else {
			t.Errorf("Lookup(%v) should return %v, found %v", tt.in, tt.out, found)
		}
	}
}

func TestGeoIP2ISP(t *testing.T) {
	initGlobal("t/etc")

	m, err := NewGeoIP2("/var/lib/GeoIP/GeoIP2-ISP.mmdb")
	if err != nil {
		t.Errorf("Error loading data: %v", err)
		return
	}

	for _, tt := range tableTestGeoIP2ISP {
		// tt.in tt.out
		record, err := m.ISP(tt.in)

		// tt.in tt.out
		if err != nil {
			t.Fatal(err)
		}
		found := record.ISP
		if found == tt.out {
			t.Logf("Lookup(%v) good", tt.in)
		} else {
			t.Errorf("Lookup(%v) should return %v, found %v", tt.in, tt.out, found)
		}
	}
}

func TestGeoIP2Country(t *testing.T) {
	initGlobal("t/etc")

	m, err := NewGeoIP2("/var/lib/GeoIP/GeoIP2-Country.mmdb")
	if err != nil {
		t.Errorf("Error loading data: %v", err)
		return
	}
	for _, tt := range tableTestGeoIP2Country {
		// tt.in tt.out
		record, err := m.Country(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		found := record.Country.IsoCode
		if found == tt.out {
			t.Logf("Lookup(%v) good", tt.in)
		} else {
			t.Errorf("Lookup(%v) should return %v, found %v", tt.in, tt.out, found)
		}
	}
}

func BenchmarkGeoIP2Country(b *testing.B) {
	initGlobal("t/etc")
	m, err := NewGeoIP2("/var/lib/GeoIP/GeoIP2-Country.mmdb")
	if err != nil {
		b.Errorf("Error loading data: %v", err)
		return
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		m.Country("2001:4998::1")
	}
}

func BenchmarkGeoIP2ASN(b *testing.B) {
	initGlobal("t/etc")
	m, err := NewGeoIP2("/var/lib/GeoIP/GeoIP2-ISP.mmdb")
	if err != nil {
		b.Errorf("Error loading data: %v", err)
		return
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.SetBytes(1)
	for n := 0; n < b.N; n++ {
		m.ISP("2001:4998::1")
	}
}
