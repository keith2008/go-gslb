package main

/*
Reads and searches (in O(log n) time the MaxMind GeoLite ASN data.
We use this data to determine what ISP a given IP address belongs to.

This package depends on MaxMind's GeoLite ASN data,
published in what is considered their legacy format.

http://dev.maxmind.com/geoip/legacy/geolite/

Synopsis
    m := maxmind.New("../data/GeoIPASNum2.csv", "../data/GeoIPASNum2v6.csv")
    asn, isp := m.Lookup("2600::") // Expect 3651, Sprint

*/

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"io"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
)

// AsnInfo describes a range of IP addresses, and the ISP's BGP ASN and ISP name.
type AsnInfo struct {
	start net.IP // Always in IPv6 format, even if IPv4.
	stop  net.IP // Always in IPv6 format, even if IPv4.
	asn   string // ie "7922".  We ultimately use this as a string; saves a conversion at runtime
	isp   string // ie "Comcast"
}

// MaxMind is an ordered slice of IP ranges, describing the ISP information for known IP addresses.
type MaxMind struct {
	asnInfo    []AsnInfo
	fileV4Info FileInfoType
	fileV6Info FileInfoType
}

// byStart provides an interface for sort.Sort()
type byStart []AsnInfo

func (a byStart) Len() int           { return len(a) }
func (a byStart) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byStart) Less(i, j int) bool { return bytes.Compare(a[i].start, a[j].start) < 0 }

// NeedReload indicates if the MaxMind files should be reloaded from disk
func (m *MaxMind) NeedReload() bool {
	return FileModifiedSince(m.fileV4Info) ||
		FileModifiedSince(m.fileV6Info)
}

// loadCsvV4 loads MaxMind's "Legacy" formatted CSV for IPv4 to ASN
func (m *MaxMind) loadCsvV4(filename string) error {
	//import "encoding/csv"

	// Remember the file info for later, so we know if we need to reload
	// Order matters! stat first, then open.
	m.fileV4Info, _ = FileModifiedInfo(filename)

	// Actually open the file now.
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// start (int32), stop (int32), string with ASN and company name
	// 16809984,16810495,"AS23969 TOT Public Company Limited"

	var myRe = regexp.MustCompile(`^AS(\d+) (.*)$`)

	r := csv.NewReader(bufio.NewReader(f))
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}
		var a AsnInfo
		a.start = make(net.IP, 4)
		startInt, err := strconv.ParseUint(record[0], 10, 32)
		binary.BigEndian.PutUint32(a.start, uint32(startInt))
		a.start = a.start.To16()

		a.stop = make(net.IP, 4)
		stopInt, err := strconv.ParseUint(record[1], 10, 32)
		binary.BigEndian.PutUint32(a.stop, uint32(stopInt))
		a.stop = a.stop.To16()

		//		fmt.Printf("Start=%v, Stop=%v\n", a.start.String(), a.stop.String())

		matches := myRe.FindStringSubmatch(record[2])
		if matches != nil {
			a.asn = matches[1] + "" // Force new strings, free underlying ones
			a.isp = matches[2] + "" // Force new strings, free underlying ones
		}

		m.asnInfo = append(m.asnInfo, a) // Add new completed record

	}
	return nil
}

// loadCsvV4 loads MaxMind's "Legacy" formatted CSV for IPv6 to ASN
func (m *MaxMind) loadCsvV6(filename string) error {
	//import "encoding/csv"

	// Remember the file info for later, so we know if we need to reload
	// Order matters! stat first, then open.
	m.fileV6Info, _ = FileModifiedInfo(filename)

	// Actually open the file now.
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// string with ASN, then start, then stop
	// "AS174 Cogent Communications",2001:49f0:1::,2001:49f0:1:ffff:ffff:ffff:ffff:ffff,48

	var myRe = regexp.MustCompile(`^AS(\d+) (.*)$`)

	r := csv.NewReader(bufio.NewReader(f))
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		var a AsnInfo
		a.start = net.ParseIP(record[1])
		a.stop = net.ParseIP(record[2])

		matches := myRe.FindStringSubmatch(record[0])
		if matches != nil {
			a.asn = matches[1] + "" // Force new strings, free underlying ones
			a.isp = matches[2] + "" // Force new strings, free underlying ones
		}

		m.asnInfo = append(m.asnInfo, a) // Add new completed record

	}
	return nil
}

// NewMaxMind loads v4 and v6 MaxMind "Legacy" IP to ASN data.
// CSV files are specified.  Pass empty tryings if you don't
// want the file loaded.
// Creating this object is threadsafe; however, accessing this
// object is not.  The finished object should no be altered
// by goroutines.
func NewMaxMind(fileNameV4 string, fileNameV6 string) (*MaxMind, error) {
	m := new(MaxMind)
	var err error

	if fileNameV4 != "" {
		err = m.loadCsvV4(fileNameV4) // Modifies m.asnInfo
		if err != nil {
			return m, err
		}
	}

	if fileNameV6 != "" {
		err = m.loadCsvV6(fileNameV6) // Modifies m.asnInfo
		if err != nil {
			return m, err
		}
	}

	sort.Sort(byStart(m.asnInfo))
	return m, err
}

// Lookup returns the ASN and ISP name for a given IP address.
// If not found, returns as=0 and isp=""
// To get "m", call GlobalMaxMind().
// To do a lookup, try GlobalMaxMind().Lookup(string)
func (m MaxMind) Lookup(s string) (as string, isp string) {
	i := net.ParseIP(s).To16()
	a := m.asnInfo // Get an easy handle to the slice of AsnInfo

	// Do binary search.  "f" will be the offset found.
	f := sort.Search(len(a), func(key int) bool { return bytes.Compare(a[key].stop, i) >= 0 })
	//	fmt.Printf("lookup(%v) %v bytes\n",i,len(i));

	if f < len(a) && bytes.Compare(a[f].start, i) <= 0 && bytes.Compare(i, a[f].stop) <= 0 {
		//	     fmt.Printf("we want it\n");
		//	     fmt.Println(a[f])
		as = a[f].asn
		isp = a[f].isp
	}
	return
}
