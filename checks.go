package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Dispatch function.  Any new checks must also update this function.
func dispatchServiceCheck(service string, target string) (bool, error) {
	// Do stuff, once

	switch service {
	case "check_true":
		return checkTrue(target)
	case "check_false":
		return checkFalse(target)
	case "check_http":
		return checkHTTP(target)
	case "check_mirror":
		return checkMirror(target)
	case "check_irc":
		return checkIRC(target)

	}
	log.Printf("Unexpected service name %v, fix your configs!\n", service)
	return false, errors.New("Unexpected service name")
}

func checkTrue(url string) (bool, error) {
	return true, nil
}
func checkFalse(url string) (bool, error) {
	return false, nil
}
func checkIRC(url string) (bool, error) {
	return checkNetworkHostPort("tcp", url+":6667")
}

// LookupAddress - given a name, a view, *and* a RR type
// Returns the first matching record found (not multiple!).
// Used by health checks.
func LookupAddressByType(qname string, token string) (value string, ok bool) {
	name := qname
	zoneRef := GlobalZoneData()
	view := "default"
	token = strings.ToUpper(token)                        // Make sure this is canonicalized, just in case
	lookup := LookupBackEnd(name, view, true, 2, zoneRef) // Do we know anything about this name?
	for _, line := range lookup {
		words := QuotedStringToWords(line)
		lastword := words[len(words)-1]
		if token == strings.ToUpper(words[0]) {
			return lastword, true
		}
	}
	return qname, false // Give back the original string, maybe they'll use it. Or not.
}
func LookupAddressA(qname string) (value string, ok bool) {
	val, ok := LookupAddressByType(qname, "A")
	return val, ok
}
func LookupAddressAAAA(qname string) (value string, ok bool) {
	val, ok := LookupAddressByType(qname, "AAAA")
	return val, ok
}
func LookupAddressDS(qname string) (value string, ok bool) {
	val, ok := LookupAddressByType(qname, "AAAA")
	if !ok {
		val, ok = LookupAddressByType(qname, "A")
	}
	return val, ok
}
func LookupAddressHostPort(qname string, port string) (hostport string, ok bool) {

	// First: See if they specified host:port already; capture port
	h, p, err := net.SplitHostPort(qname)
	if err == nil {
		qname = h // Update to the new hostname
		port = p  // and replace the default port number with this
	}

	// What if the qname is really an IP address?
	ip := net.ParseIP(h)
	if ip != nil {
		return net.JoinHostPort(ip.String(), port), true
	}

	// Not an IP address. Do we know the name internally in zone.conf ?
	if val, ok := LookupAddressByType(qname, "AAAA"); ok {
		return net.JoinHostPort(val, port), true
	}
	if val, ok := LookupAddressByType(qname, "A"); ok {
		return net.JoinHostPort(val, port), true
	}

	// Not an IP address, and not an internal name.
	// Give back the external name; the caller can use DNS.
	return net.JoinHostPort(qname, port), false
}

func checkNetworkHostPort(proto string, hostport string) (bool, error) {
	seconds := 10
	timeout := time.Duration(seconds) * time.Second
	c, err := net.DialTimeout(proto, hostport, timeout)
	if err != nil {
		return false, err
	}
	c.Close()
	return true, nil
}

// check_http will always do port 80.
func checkHTTP(host string) (bool, error) {
	// Allow up to 10 seconds to try this out.
	return checkHTTPHelper(host, "80")
}

// checkHTTPHelper will check any port, not just 80.
func checkHTTPHelper(host string, port string) (bool, error) {
	// Allow up to 10 seconds to try this out.
	seconds := 10
	timeout := time.Duration(seconds) * time.Second
	client := &http.Client{Timeout: timeout}
	hostport, _ := LookupAddressHostPort(host, port) // 192.0.2.1:80 or [2001:db8::1]:80
	url := "http://" + hostport

	req, err := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	if resp.StatusCode/100 == 2 {
		return true, nil

	}

	err = fmt.Errorf("url %s http status %v", host, resp.StatusCode)
	return false, err

}

func checkMirror(hostname string) (bool, error) {
	b, err := checkMirrorHelper(hostname) // Check the named site first
	if err == nil && b == true {
		b, err = checkMirrorHelper("mtu1280." + hostname) // Check also implied mtu1280.site as well
	}
	return b, err
}

func checkMirrorHelper(hostname string) (bool, error) {
	hostport, _ := LookupAddressHostPort(hostname, "80") // Avoid external DNS lookups
	url := "http://" + hostport + "/site/config.js"      // Mirrors should have this file
	seconds := 10                                        // Allow up to 10 seconds to try this out.
	timeout := time.Duration(seconds) * time.Second      // Nothing is ever easy
	client := &http.Client{Timeout: timeout}             // Create the client configuration object

	req, err := http.NewRequest("GET", url, nil) // Create the client request object
	req.Host = "test-ipv6.com"                   // Override the "Host:" field
	resp, err := client.Do(req)                  // Actually perform the GET
	if err != nil {
		return false, err // Return the error
	}
	defer resp.Body.Close()                                     // If the request worked, one MUST ALWAYS close the body.  ALWAYS.
	body, err := ioutil.ReadAll(resp.Body)                      // Grab the body content.
	b := strings.Contains(string(body), `master.test-ipv6.com`) // Check to see if it looks like our master site is mentioned.
	return b, nil                                               // Return true/false, no errors.
}
