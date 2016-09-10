package main

import (
	"log"
	"strings"

	"github.com/miekg/dns"
)

/*

Starts a DNS server, on the network addresses specified in the config file.
This is done only once, on startup -  we don't have a mechanism at
this time to shut down DNS listeners.  This means, any changes to the
server config relating to IP addresses and ports should be followed by
a server restart.

This in server.conf will start DNS servers on port 53, both IPv4 and IPV6, UDP and TCP.

[server]
udp: [::]:53
tcp: [::]:53

*/

// initDNS will configure handlers to respond to specific DNS hostnames with our
// "special" names; as well as the general purpose handleGSLB for everything
// else.  It will be handleGSLB's responsibility  to refuse service.
func initDNS() {
	initDNSSpecialHandlers("ip", handleIP)           // Might have more than one name for specialty responders
	initDNSSpecialHandlers("as", handleAS)           // Might have more than one name for specialty responders
	initDNSSpecialHandlers("isp", handleISP)         // Might have more than one name for specialty responders
	initDNSSpecialHandlers("country", handleCountry) // Might have more than one name for specialty responders
	initDNSSpecialHandlers("view", handleView)       // Might have more than one name for specialty responders
	initDNSSpecialHandlers("maxmind", handleMaxMind) // Might have more than one name for specialty responders
	initDNSSpecialHandlers("help", handleHelp)       // Might have more than one name for specialty responders
	initDNSSpecialHandlers("break", handleBreak)     // Might have more than one name for specialty responders
	dns.HandleFunc(".", handleGSLB)                  // Anything else, send it to the heavier weight processor.
}

// initDNSSpecialHandlers will look in your config for a named config variable,
// and attach the specified DNS handler.  This is to make initDNS() less tedious.
func initDNSSpecialHandlers(name string, handler func(dns.ResponseWriter, *dns.Msg)) {
	c := GlobalConfig() // Get our config object
	s, _ := c.GetSectionNameValueStrings("special", name)
	if s != nil {
		for _, dom := range s {
			pattern := dom
			if !(strings.HasSuffix(pattern, ".")) {
				pattern = pattern + "."
			}
			Debugf("### Registering %s for handler %#v\n", pattern, handler)
			dns.HandleFunc(pattern, handler)
		}
	}
}

// Start DNS services.
func startDNS() {
	initDNS()

	counter := 0 // Keep a count of how many DNS servers we found.

	c := GlobalConfig()                            // Get our config object
	for _, proto := range []string{"udp", "tcp"} { // Checing for "tcp" and "udp"
		if addrs, ok := c.GetSectionNameValueStrings("server", proto); ok {
			counter++                    // Note how many we found
			for _, addr := range addrs { // For every address specified
				server := &dns.Server{Addr: addr, Net: proto} // createa a server configuration
				go func() {                                   // And background the listener.
					err := server.ListenAndServe()
					if err != nil {
						log.Fatalf("startDNS: Failed to start server.  Error: %s Parameters: %#v\n", err.Error(), server)
					}
				}()
			}

		}
	}

	if counter == 0 {
		log.Fatalf("StartDNS: Failed to find server for tcp or udp\n")
	}
}
