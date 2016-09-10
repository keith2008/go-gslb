package main

/*
This file contains global variables that are used by various functions.
There are functions used for making changes to those variables; as well
as general "go reload all the things" functions.

Access to global variables MUST use the provided functions to be given threadsafe access.

Everything inside Global is considered read-only when set.
Any writes should be done before assigning the structure to the global objects.

IE: Fully load and parse configs, before setting the Global.Config variable.
Anyone can (at any time) get the latest Global.Config, and start referencing the pointer for read access.

Funtions which grab a pointer should periodically refresh; this gives old structures a chance to get garbage collected, and for new configs to take effect.


*/

import (

	//	"github.com/davecgh/go-spew/spew"

	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// GlobalStruct is a container for our global variables.
type GlobalStruct struct {
	Config        atomic.Value // server.conf: all server configs
	ZoneData      atomic.Value // zone.conf: Read from disk; drives isp and also healthchecks
	ViewData      atomic.Value // dynamic: ASN to ISP and Resolver to ISP lookups
	GeoIP2Country atomic.Value // GeoIP2
	GeoIP2ISP     atomic.Value // GeoIP2
}

// Global is a container for our global variables.
var Global GlobalStruct
var initOnce sync.Once

// initGlobal is called by main (and by unit tests) to start
// empty configs before anyone tries to use the maps.
// Calls LoadConfigs to get real config data.
// Sets up a background scanner to detect changed configs.
func initGlobal(etc string) {
	// Quick init everything
	onceBody := func() {

		log.Printf("initGlobal(%v)\n", etc)
		SetGlobalConfig(NewConfig())
		SetGlobalZoneData(NewConfig())
		SetGlobalViewData(NewConfig())
		if m, err := NewGeoIP2(""); err == nil {
			SetGlobalGeoIP2ISP(m)
		}
		if m, err := NewGeoIP2(""); err == nil {
			SetGlobalGeoIP2Country(m)
		}

		LoadConfigs(etc)
		go taskScanConfigs(etc)
	}
	initOnce.Do(onceBody)

}

// SetGlobalConfig safely sets the *Config object (threadsafe)
func SetGlobalConfig(c *Config) {
	Global.Config.Store(c)
}

// GlobalConfig returns the current configuration *Config object.
// Once acquired, you can safely use that object for RO operations.
func GlobalConfig() *Config {
	return Global.Config.Load().(*Config)
}

// GlobalConfigAvailable indicates that the global config
// is ready to use
func GlobalConfigAvailable() bool {
	i := Global.Config.Load()
	return i != nil
}

// SetGlobalZoneData safely sets the *Config object (threadsafe)
func SetGlobalZoneData(c *Config) {
	Global.ZoneData.Store(c)
}

// GlobalZoneData safely sets the zone data  *Config object.
// Once acquired, you can safely use that object for RO operations.
func GlobalZoneData() *Config {
	return Global.ZoneData.Load().(*Config)
}

// SetGlobalViewData sets the new configuration *Config object (threadsafe)
func SetGlobalViewData(c *Config) {
	Global.ViewData.Store(c)
}

// GlobalViewData returns the current zone data  *Config object.
// Once acquired, you can safely use that object for RO operations.
func GlobalViewData() *Config {
	return Global.ViewData.Load().(*Config)
}

// SetGlobalGeoIP2Country sets the new configuration *Config object (threadsafe)
func SetGlobalGeoIP2Country(m *GeoIP2) {
	Global.GeoIP2Country.Store(m)
}

// GlobalGeoIP2Country ...
func GlobalGeoIP2Country() *GeoIP2 {
	return Global.GeoIP2Country.Load().(*GeoIP2)
}

// SetGlobalGeoIP2ISP sets the new configuration *Config object (threadsafe)
func SetGlobalGeoIP2ISP(m *GeoIP2) {
	Global.GeoIP2ISP.Store(m)
}

// GlobalGeoIP2ISP ...
func GlobalGeoIP2ISP() *GeoIP2 {
	return Global.GeoIP2ISP.Load().(*GeoIP2)
}

// LoadConfigs will re-read all configs, as well as flush query caches.
func LoadConfigs(path string) {
	log.Printf("LoadConfigs(%v)\n", path)

	loadConfig(path + "/server.conf") // Latest server config object
	loadZone(path + "/zone.conf")
	loadGeoIP2Country("/var/lib/GeoIP/GeoIP2-Country.mmdb") // Used for Country ISO
	loadGeoIP2ISP("/var/lib/GeoIP/GeoIP2-ISP.mmdb")         // Used for ASN and ISP name
	scanForHealthChecks()                                   // Starts new background checks if needed
	ClearCaches("Configuration files loaded")               // Flush any and all caches after any config has changed
}

// scanConfigs Check to see if we need to reload anything.
func scanConfigs(etc string) {

	// Get pointers to current active versions
	m1 := GlobalGeoIP2Country()
	m2 := GlobalGeoIP2ISP()
	c := GlobalConfig()
	z := GlobalZoneData()

	//	fmt.Printf("scanConfigs() trace info m=%v c=%vv z=%v\n", m.NeedReload(), c.NeedReload(), z.NeedReload())

	if m1.NeedReload() ||
		m2.NeedReload() ||
		c.NeedReload() ||
		z.NeedReload() {
		Debugf("LoadConfigs()\n")
		LoadConfigs(etc) // This will change Global.* pointers to new versions
	}
}

func loadConfig(path string) {

	Debugf("loadConfig(%v)\n", path)

	C, err := NewConfigFromFile(path)
	if err != nil {
		log.Fatalf("Fatal error loading %v: %v\n", path, err)
	}
	SetGlobalConfig(C) // Safely store latest finished product into global
}

func loadZone(path string) {
	Debugf("loadZone(%v)\n", path)
	C, err := NewConfigFromFile(path)
	if err != nil {
		log.Fatalf("Fatal error loading %v: %v\n", path, err)
	}
	SetGlobalZoneData(C)
	scanForASN()
}
func loadGeoIP2Country(filename string) {
	M, err := NewGeoIP2(filename)
	if err != nil {
		log.Printf("ERROR: Failed to load MaxMind data %v: %v", filename, err.Error())
	} else {
		SetGlobalGeoIP2Country(M)
	}
}
func loadGeoIP2ISP(filename string) {
	M, err := NewGeoIP2(filename)
	if err != nil {
		log.Printf("ERROR: Failed to load MaxMind data %v: %v", filename, err.Error())
	} else {
		SetGlobalGeoIP2ISP(M)
	}
}

func taskScanConfigs(etc string) {
	for {
		scanConfigs(etc)
		c := GlobalConfig() // Get latest active config object
		sleepsecs := int(1) // Default, in case we can't find a suitable sleep
		if sleepsecsStr, ok := c.GetSectionNameValueString("interval", "configs"); ok {
			sleepsecs, _ = strconv.Atoi(sleepsecsStr)
		}
		time.Sleep(time.Duration(sleepsecs) * time.Second)
	}
}

func scanForHealthChecks() {
	z := GlobalZoneData() // Safely copy a pointer to latest
	c := GlobalConfig()   // Safely copy a pointer to latest

	// Need to read all the data, see what health checks are needed
	for key, val := range z.Data {
		//		fmt.Printf("key=%v val=%v\n", key, val)
		for _, s := range val.Values {
			words := strings.Fields(s)
			if words[0] == "HC" {
				if false {
					Debugf("key=%v check=%v name=%v\n", key, words[1], words[2])
				}
				service := words[1]
				target := words[2]

				sleepsecs := int(30) // fallback
				if sleepsecsStr, ok := c.GetSectionNameValueString("interval", service); ok {
					sleepsecs, _ = strconv.Atoi(sleepsecsStr)
				}
				AddCheck(service, target, sleepsecs)
			}
		}
	}
}

func scanForASN() {
	z := GlobalZoneData() // Copy a pointer now, in case Global.Zone wants to change later

	I := NewConfig() // Store ASN lookups here

	// Need to read all the data, see what health checks are needed
	for key, val := range z.Data {
		if (key.Name == "country" || key.Name == "as") || (key.Name == "resolver") {
			for _, s := range val.Values {
				//  s = the resolver or the AS number
				I.AddKeyValue(ConfigKey{"default", s}, key.Section) // Adding {default/7922} Comcast
			}
		}
	}
	SetGlobalViewData(I) // Replace the previous lookup table with a new one.
}
