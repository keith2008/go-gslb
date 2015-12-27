package main

import "testing"

func TestConfigSwapped(t *testing.T) {

	initGlobal("t/etc")

	// Make sure that if we load a new config,
	// it is actually made available to callers
	// calling GlobalConfig().  Do this by
	// testing our normal config file but loading
	// it with a "./" prefixed name to make it
	// different.

	loadConfig("./t/etc/server.conf") // That path is intentional.
	c1 := GlobalConfig()
	loadConfig("t/etc/server.conf") // And back to the original path.

	if c1.FileInfo.Name == "./t/etc/server.conf" {
		t.Log("Successfully reloaded ./t/etc/server.conf and set it global")
	} else {
		t.Errorf("Global config not reloaded (or setting it global failed)")
	}

}

func Benchmark_GlobalConfig(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")

	b.ResetTimer()
	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		_ = GlobalConfig()
	}
}

func Benchmark_SetGlobalConfig(b *testing.B) {
	// Expensive stuff first
	initGlobal("t/etc")
	c := GlobalConfig()

	b.ResetTimer()
	// Now loop the important part of the benchmark
	for n := 0; n < b.N; n++ {
		SetGlobalConfig(c)
	}
}
