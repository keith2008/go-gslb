// Our example hello world
package main

// https://golang.org/pkg/sort/

import (
	"strconv"
	"testing"
)

var tableTestMaxMindASN = []struct {
	in  string
	out string
}{
	{"50.184.213.245", "7922"},
	{"2001:4998::1", "10310"},
}

var tableTestMaxMindGeoISP = []struct {
	in  string
	out string
}{
	{"50.184.213.245", "Comcast Cable Communications, Inc."},
	{"2001:4998::1", "Yahoo!"},
}

func TestMaxMindASN(t *testing.T) {
	initGlobal("t/etc")

	m, err := NewMaxMind("t/etc/GeoIPASNum2.csv", "t/etc/GeoIPASNum2v6.csv")
	if err != nil {
		t.Errorf("Error loading data: %v", err)
		return
	}
	for _, tt := range tableTestMaxMindASN {
		// tt.in tt.out
		asn, _ := m.LookupAsnPlusName(tt.in)
		asnString := strconv.FormatUint(uint64(asn), 10)
		if asnString == tt.out {
			t.Logf("Lookup(%v) good", tt.in)
		} else {
			t.Errorf("Lookup(%v) should return %v, found %v", tt.in, tt.out, asn)
		}
	}
}

func TestMaxMindGeoISP(t *testing.T) {
	initGlobal("t/etc")

	m, err := NewMaxMind("t/etc/GeoIPASNum2.csv", "t/etc/GeoIPASNum2v6.csv")
	if err != nil {
		t.Errorf("Error loading data: %v", err)
		return
	}
	for _, tt := range tableTestMaxMindGeoISP {
		// tt.in tt.out
		_, isp := m.LookupAsnPlusName(tt.in)
		if isp == tt.out {
			t.Logf("Lookup(%v) good", tt.in)
		} else {
			t.Errorf("Lookup(%v) should return %v, found %v", tt.in, tt.out, isp)
		}
	}
}

func BenchmarkMaxMindLookupAsn(b *testing.B) {
	initGlobal("t/etc")

	m, _ := NewMaxMind("t/GeoIPASNum2.csv", "t/GeoIPASNum2v6.csv")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = m.LookupAsn("2001:4998::1")
	}
}

func BenchmarkMaxMindLookupAsnPlusName(b *testing.B) {
	initGlobal("t/etc")

	m, _ := NewMaxMind("t/GeoIPASNum2.csv", "t/GeoIPASNum2v6.csv")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		asn, _ := m.LookupAsnPlusName("2001:4998::1")
		_ = strconv.FormatUint(uint64(asn), 10)

	}
}
