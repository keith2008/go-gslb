package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

var checksInitOnce sync.Once
var WebServerHostPort = "" // Will populate on init

// FakeMirrorJsConfig produces a simple respose, with the magic text that check_mirror wants
func FakeMirrorJsConfig(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, "mocked web server for testing.\nmaster.test-ipv6.com\n")
}

// FakeWebServer starts a web server on localhost, on a random port. Returns the port number.
// As part of this, a handler is set up for /site/config.js .
func FakeWebServer(t *testing.T) string {
	if WebServerHostPort == "" {
		listner, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal("Listen:", err)
		}

		WebServerHostPort = listner.Addr().String()
		bg := func() {
			err2 := http.Serve(listner, nil)
			if err2 != nil {
				log.Fatal("http.Serve:", err2)
			}
		}

		go bg()
	}

	return WebServerHostPort
}

// TestWebServer is to make sure we can start the web server.
// I don't want to chase problems with health checks
// if the real problem is I can't start a web server.
func TestWebServer(t *testing.T) {
	FakeWebServer(t)
	log.Printf("FakeWebServer(): %v\n", WebServerHostPort)
}

// TestCheckHttpDown tests http for 404
func TestCheckHttpDown(t *testing.T) {
	initGlobal("t/etc")
	FakeWebServer(t)

	b, err := checkHTTP(WebServerHostPort)
	if b != true {
		t.Logf("checkHTTP(%s) good", WebServerHostPort)
	} else {
		t.Errorf("checkHTTP(%s) error: %v", WebServerHostPort, err)
	}
}

// TestCheckHttpUp tests http for 200
func TestCheckHttpUp(t *testing.T) {
	initGlobal("t/etc")
	FakeWebServer(t)
	http.HandleFunc("/", FakeMirrorJsConfig)

	b, err := checkHTTP(WebServerHostPort)
	if b == true {
		t.Logf("checkHTTP(%s) good", WebServerHostPort)
	} else {
		t.Errorf("checkHTTP(%s) error: %v", WebServerHostPort, err)
	}
}

// TestCheckMirror checks /site/config.js for "master.test-ipv6.com" in the content
func TestCheckMirrorHelper(t *testing.T) {
	initGlobal("t/etc")
	FakeWebServer(t)
	http.HandleFunc("/site/config.js", FakeMirrorJsConfig)

	b, err := checkMirrorHelper(WebServerHostPort)
	if b == true {
		t.Logf("checkMirrorHelper(%s) good", WebServerHostPort)
	} else {
		t.Errorf("checkMirrorHelper(%s) error: %v", WebServerHostPort, err)
	}
}

func TestCheckHttpLiteral(t *testing.T) {

	url := "http://[2001:470:1:18::119]/ip/"

	seconds := 10
	timeout := time.Duration(seconds) * time.Second
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest("GET", url, nil)
	req.Host = "ipv6.test-ipv6.com"
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Failed %s %v", url, err)
		return
	}
	defer resp.Body.Close()
	_, _ = ioutil.ReadAll(resp.Body)
	t.Logf("Success %v url %s", resp.StatusCode, url)

}

var tableLookupLookupAddressHostPort = []struct {
	name string
	port string
	val  string
	ok   bool
}{
	{"a.example.com", "80", "192.0.2.1:80", true},
	{"aaaa.example.com", "80", "[2001:db8::1]:80", true},
	{"ds.example.com", "80", "[2001:db8::1]:80", true},
	{"expand.example.com", "80", "[2001:db8::1]:80", true},
	{"192.0.2.1:80", "80", "192.0.2.1:80", true},
	{"192.0.2.1:8080", "80", "192.0.2.1:8080", true},
	{"[2001:db8::1]:80", "80", "[2001:db8::1]:80", true},
	{"[2001:db8::1]:8080", "80", "[2001:db8::1]:8080", true},
	{"offhost.example.net", "80", "offhost.example.net:80", false},
	{"offhost.example.net:8080", "80", "offhost.example.net:8080", false},
}

func TestLookupLookupAddressHostPort(t *testing.T) {

	for _, tt := range tableLookupLookupAddressHostPort {
		val, ok := LookupAddressHostPort(tt.name, tt.port)
		if val != tt.val || ok != tt.ok {
			t.Errorf("LookupAddressHostPort(%v,%v) expected (%v,%v) found (%v,%v)", tt.name, tt.port, tt.val, tt.ok, val, ok)
		} else {
			t.Logf("LookupAddressHostPort(%v,%v) ok", tt.name, tt.port)
		}
	}
}
