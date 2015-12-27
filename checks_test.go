package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
)

var checksInitOnce sync.Once
var WebServerHostPort = "" // Will populate on init

// FakeMirrorJsConfig produces a simple respose, with the magic text that check_mirror wants
func FakeMirrorJsConfig(w http.ResponseWriter, req *http.Request) {
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
		t.Logf("check_http(%s) good", WebServerHostPort)
	} else {
		t.Errorf("check_http(%s) error: %v", WebServerHostPort, err)
	}
}

// TestCheckHttpUp tests http for 200
func TestCheckHttpUp(t *testing.T) {
	initGlobal("t/etc")
	FakeWebServer(t)
	http.HandleFunc("/", FakeMirrorJsConfig)

	b, err := checkHTTP(WebServerHostPort)
	if b == true {
		t.Logf("check_http(%s) good", WebServerHostPort)
	} else {
		t.Errorf("check_http(%s) error: %v", WebServerHostPort, err)
	}
}

// TestCheckMirror checks /site/config.js for "master.test-ipv6.com" in the content
func TestCheckMirror(t *testing.T) {
	initGlobal("t/etc")
	FakeWebServer(t)
	http.HandleFunc("/site/config.js", FakeMirrorJsConfig)

	b, err := checkMirror(WebServerHostPort)
	if b == true {
		t.Logf("check_mirror(%s) good", WebServerHostPort)
	} else {
		t.Errorf("check_mirror(%s) error: %v", WebServerHostPort, err)
	}
}
