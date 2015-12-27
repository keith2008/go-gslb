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

func checkHTTP(host string) (bool, error) {
	// Allow up to 10 seconds to try this out.
	seconds := 10
	timeout := time.Duration(seconds) * time.Second
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest("GET", "http://"+host, nil)
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
	url := "http://" + hostname + "/site/config.js"
	// Allow up to 10 seconds to try this out.
	seconds := 10
	timeout := time.Duration(seconds) * time.Second
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest("GET", url, nil)
	req.Host = "test-ipv6.com"
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	b := strings.Contains(string(body), `master.test-ipv6.com`)
	return b, nil
}
