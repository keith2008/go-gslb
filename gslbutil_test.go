// Our example hello world

package main

import (
	"fmt"
	"testing"
)

var tableTestQuotedStringToWords = []struct {
	in  string
	out string
}{
	{"a b \"c d\"", `[]string{"a", "b", "\"c d\""}`},
	{"a b 'c d'", `[]string{"a", "b", "'c d'"}`},
}

func TestQuotedStringToWords(t *testing.T) {
	initGlobal("t/etc")

	// Make sure that the interface to QuotedStringToWords has not broken.
	// One obnoxious behavior it has, is that quoted words and strings.. remain quoted
	// Since other parts of our app are now expecting it, we need to keep an eye on it.
	for _, tt := range tableTestQuotedStringToWords {
		// tt.in tt.out
		res := QuotedStringToWords(tt.in)
		found := fmt.Sprintf("%#v", res)
		if found == tt.out {
			t.Logf("QuotedStringToWords(%v) good", tt.in)
		} else {
			t.Errorf("QuotedStringToWords(%v) should return %s, found %s", tt.in, tt.out, found)
		}
	}
}

func BenchmarkQuotedStringToWords(b *testing.B) {
	initGlobal("t/etc")

	// Expensive stuff first
	b.ResetTimer()

	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = QuotedStringToWords(tableTestQuotedStringToWords[0].in)
	}
}
