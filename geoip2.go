package main

/*

Reads MaxMind GeoIP2 databases.

This package depends on MaxMind's GeoLite ASN data,
published in what is considered their legacy format.

http://dev.geoip2.com/geoip/legacy/geolite/

Synopsis
    m := geoip2.New("../data/GeoIPASNum2.csv", "../data/GeoIPASNum2v6.csv")
    asn, isp := m.Lookup("2600::") // Expect 3651, Sprint

*/

import (
	"errors"
	"log"
	"net"
	"runtime"

	"github.com/oschwald/geoip2-golang"
)

// ErrBadIP is returned if the IP address is not parseable
var ErrBadIP = errors.New("Bad IP address")

// ErrNotLoaded is returned if the IP address is not parseable
var ErrNotLoaded = errors.New("data not yet loaded")

// GeoIP2 is a convenience struct for us to manage
// the geoip2 resources
type GeoIP2 struct {
	handle   *geoip2.Reader
	fileInfo FileInfoType
	lookup   func(string) (string, error)
}

// NeedReload indicates if the MaxMind files should be reloaded from disk
func (m *GeoIP2) NeedReload() bool {
	return FileModifiedSince(m.fileInfo)
}

// Close will shut down the current handle.
func (m *GeoIP2) Close() {
	if m.handle != nil {
		m.handle.Close()
		m.handle = nil
	}
}

// Handle gets the geoip2 Reader handle
// providing more direct access to the various lookup
// capabilities.
func (m *GeoIP2) Handle() *geoip2.Reader {
	return m.handle
}

// Country lookup by IP
func (m *GeoIP2) Country(ipstring string) (*geoip2.Country, error) {
	ip := net.ParseIP(ipstring)
	if ip == nil {
		return nil, ErrBadIP
	}
	if m.handle == nil {
		return nil, ErrNotLoaded
	}
	record, err := m.handle.Country(ip)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// ISP lookup by IP
func (m *GeoIP2) ISP(ipstring string) (*geoip2.ISP, error) {
	ip := net.ParseIP(ipstring)
	if ip == nil {
		return nil, ErrBadIP
	}
	if m.handle == nil {
		return nil, ErrNotLoaded
	}
	record, err := m.handle.ISP(ip)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// NewGeoIP2 loads a MaxMind GeoIP2 database, returns
// a convience handle common to the gslb project.
func NewGeoIP2(fileName string) (*GeoIP2, error) {
	m := new(GeoIP2)

	// If we ever go out of scope, and GC wnats to clean up...
	// we want to auto-close the database handle.
	// We expect this to happen any time we see a new database
	// and hot-swap the handle from old to new, leavig the old
	// open for any in-flight queries.
	runtime.SetFinalizer(m, func(o *GeoIP2) {
		log.Printf("Closing an old GeoIP instance\n")
		o.Close()
	})

	// no file? no problem.
	if fileName == "" {
		return m, nil
	}

	m.fileInfo, _ = FileModifiedInfo(fileName)

	var err error
	m.handle, err = geoip2.Open(fileName)
	if err != nil {
		return m, err
	}
	return m, err
}
