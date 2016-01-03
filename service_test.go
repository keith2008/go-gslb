package main

import (
	"testing"
	"time"
)

func TestBackgroundServiceChecks(t *testing.T) {
	initGlobal("t/etc")
	status := true

	// Add, make sure it doesn't yet exist.
	status = AddCheck("check_true", "gigo.com", 1)
	if status != false {
		t.Fatalf("AddCheck claims to have already started check_true / gigo.com")
	}
	t.Log("AddCheck(check_true, gigo.com) good")

	// Add, make sure it did already exist (nearly a no-op)
	status = AddCheck("check_true", "gigo.com", 1)
	if status != true {
		t.Fatalf("AddCheck claims we never started check_true / gigo.com")
	}
	t.Log("AddCheck(check_true, gigo.com) still good")

	status = AddCheck("check_false", "gigo.com", 1)
	status = AddCheck("check_false", "gigo.com", 1)

	// Give it a chance to health check.
	time.Sleep(time.Duration(2) * time.Second)

	// Investigate health status.
	status, _ = GetStatus("check_true", "gigo.com")
	if status != true {
		t.Fatalf("GetStatus(check_true,gigo.com) not yet true")
	}
	t.Log("GetStatus(check_true, gigo.com) good")

	status, _ = GetStatus("check_false", "gigo.com")
	if status != false {
		t.Fatalf("GetStatus(check_false,gigo.com) not actually false")
	}
	t.Log("GetStatus(check_false, gigo.com) good")

}

func BenchmarkSetStatus(b *testing.B) {
	initGlobal("t/etc")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		SetStatus("x", "y", true)
	}
}
func BenchmarkGetStatus(b *testing.B) {
	initGlobal("t/etc")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = GetStatus("x", "y")
	}
}

func BenchmarkEmptyFunction(b *testing.B) {
	initGlobal("t/etc")
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		empty("x", "y", true)
	}
}
