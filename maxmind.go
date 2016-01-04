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
	"encoding/csv"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

// AsnInfo describes a range of IP addresses, and the ISP's BGP ASN and ISP name.
type AsnInfo struct {
	start        [16]byte // Always in IPv6 format, even if IPv4.
	stop         [16]byte // Always in IPv6 format, even if IPv4.
	asnInt       uint32
	ispNameStart int // Offset to *MaxMind.ispNameSlice
	ispNameStop  int // Offset to *MaxMind.ispNameSlice
}

// MaxMind is an ordered slice of IP ranges, describing the ISP information for known IP addresses.
type MaxMind struct {
	asnInfo      []AsnInfo
	fileV4Info   FileInfoType
	fileV6Info   FileInfoType
	ispName      bytes.Buffer // We will populate this as we read the data files
	ispNameSlice []byte       // And update ispNameSlice to the same as ispName.Bytes()
}

// byStart provides an interface for sort.Sort()
type byStart []AsnInfo

func (a byStart) Len() int      { return len(a) }
func (a byStart) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byStart) Less(i, j int) bool {
	return bytes.Compare(a[i].start[0:], a[j].start[0:]) < 0
}

// NeedReload indicates if the MaxMind files should be reloaded from disk
func (m *MaxMind) NeedReload() bool {
	return FileModifiedSince(m.fileV4Info) ||
		FileModifiedSince(m.fileV6Info)
}

func parseMaxMindProvider(s string) (asn uint32, name string) {

	ispInfo := strings.SplitN(s, " ", 2)
	if len(ispInfo) < 1 {
		ispInfo = append(ispInfo, "AS0")
	}
	if len(ispInfo) < 2 {
		ispInfo = append(ispInfo, "Unspecified")
	}
	asString := ispInfo[0]
	if asString[0:3] == "AS " {
		asString = asString[3:]
	} else if asString[0:2] == "AS" {
		asString = asString[2:]
	}
	asInt, _ := strconv.ParseUint(asString, 10, 32)
	return uint32(asInt), ispInfo[1]
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

	r := csv.NewReader(bufio.NewReader(f))
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}
		var a AsnInfo

		startInt, err := strconv.ParseUint(record[0], 10, 32)
		startIP := net.IP{
			byte(startInt >> 24),
			byte(startInt >> 16),
			byte(startInt >> 8),
			byte(startInt)}.To16()

		stopInt, err := strconv.ParseUint(record[1], 10, 32)
		stopIP := net.IP{
			byte(stopInt >> 24),
			byte(stopInt >> 16),
			byte(stopInt >> 8),
			byte(stopInt)}.To16()

		// Copy the bytes.  Not slices.
		// We do not want memory references to the original backing (the csv file).
		for i := 0; i < 16; i++ {
			a.start[i] = startIP[i]
			a.stop[i] = stopIP[i]
		}
		asnInt, ispName := parseMaxMindProvider(record[2])
		a.asnInt = asnInt

		// ISP name record
		// Copy the bytes.  Not slices.
		// We do not want memory references to the original backing (the csv file).
		a.ispNameStart = m.ispName.Len() // Offsets instead of pointers
		m.ispName.Write([]byte(ispName)) // Effectively, appends.
		a.ispNameStop = m.ispName.Len()  // Offsets instead of pointers

		//	Adopt our new record.
		m.asnInfo = append(m.asnInfo, a) // Add new completed record

	}
	m.ispNameSlice = m.ispName.Bytes() // In case we want to read from this later!
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

	r := csv.NewReader(bufio.NewReader(f))
	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		var a AsnInfo
		startIP := net.ParseIP(record[1])
		stopIP := net.ParseIP(record[2])

		// Copy the bytes.  Not slices.
		// We do not want memory references to the original backing (the csv file).
		for i := 0; i < 16; i++ {
			a.start[i] = startIP[i]
			a.stop[i] = stopIP[i]
		}
		asnInt, ispName := parseMaxMindProvider(record[0])
		a.asnInt = asnInt

		// ISP name record
		// Copy the bytes.  Not slices.
		// We do not want memory references to the original backing (the csv file).
		a.ispNameStart = m.ispName.Len() // Offsets instead of pointers
		m.ispName.Write([]byte(ispName)) // Effectively, appends.
		a.ispNameStop = m.ispName.Len()  // Offsets instead of pointers

		//	Adopt our new record.
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
	m.asnInfo = make([]AsnInfo, 500000)

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

// LookupAsn returns the ASN for a given IP address.
// If not found, returns as=0
// To get "m", call GlobalMaxMind().
// To do a lookup, try GlobalMaxMind().LookupAsn(string)
func (m MaxMind) LookupAsn(s string) (as uint32) {
	i := net.ParseIP(s).To16()
	a := m.asnInfo // Get an easy handle to the slice of AsnInfo

	// Do binary search.  "f" will be the offset found.
	f := sort.Search(len(a), func(key int) bool { return bytes.Compare(a[key].stop[:], i) >= 0 })

	if f < len(a) && bytes.Compare(a[f].start[:], i) <= 0 && bytes.Compare(i, a[f].stop[:]) <= 0 {
		as = a[f].asnInt
	}
	return
}

// LookupAsnPlusName returns the ASN and ISP name for a given IP address.
// If not found, returns as=0 and isp=""
// To get "m", call GlobalMaxMind().
// To do a lookup, try GlobalMaxMind().LookupAsnPlusName(string)
func (m MaxMind) LookupAsnPlusName(s string) (as uint32, isp string) {
	i := net.ParseIP(s).To16()
	a := m.asnInfo // Get an easy handle to the slice of AsnInfo

	// Do binary search.  "f" will be the offset found.
	f := sort.Search(len(a), func(key int) bool { return bytes.Compare(a[key].stop[:], i) >= 0 })
	//	fmt.Printf("lookup(%v) %v bytes\n",i,len(i));
	// log.Printf("LookupAsnPlusName(%s) f=%#v\n", s, f)
	// log.Printf("%#v", a[f])

	if f < len(a) && bytes.Compare(a[f].start[:], i) <= 0 && bytes.Compare(i, a[f].stop[:]) <= 0 {
		as = a[f].asnInt
		isp = string(m.ispNameSlice[a[f].ispNameStart:a[f].ispNameStop])
		//	log.Printf("isp name is %s\n", isp)
	}
	return
}
