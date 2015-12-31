package main

import (
	"log"
	"net"
	"strings"

	"github.com/miekg/dns"
)

func findView(ipString string) (view string, asnString string, ispString string) {
	//fmt.Printf("findView(%s)\n", ipString)
	ip, _, err := net.SplitHostPort(ipString)
	if err == nil {
		ipString = ip // With the :portnumber removed.
	}

	asnString, ispString = GlobalMaxMind().Lookup(ipString) // AS number and ISP text Name

	statsMaxMind.Increment(asnString) // Keep track of queries from various service providers.

	view = "default"      // Default view name.  May override based on ASN or Resolver
	I := GlobalViewData() // Get and keep a stable (threadsafe) handle
	if found, ok := I.GetSectionNameValueString("default", asnString); ok {
		view = found
	}
	if found, ok := I.GetSectionNameValueString("default", ipString); ok {
		view = found
	}
	return view, asnString, ispString
}

// ourNewRR combined dns.NewRR with a local cache.
func ourNewRR(s string) (dns.RR, error) {

	// If needed, calculate and cache.

	// ALWAYS return a deep copy - and leave
	// the original pristine in the cache.
	// This is critical to avoid leaking
	// old queries to new (related to
	// the vixie 0x20 hack)

	if found, ok := getRRCache(s); ok {
		deep := dns.Copy(found)
		return deep, nil

	}
	// Even a fresh instance will be deep copied.
	parsed, err := dns.NewRR(s)
	if err == nil {
		setRRCache(s, parsed)
	}
	deep := dns.Copy(parsed)
	return deep, err
}

// handleReflectIP responds with the caller's IP address,
// in the form of A/AAAA as well as TXT
func handleGSLB(w dns.ResponseWriter, r *dns.Msg) {

	// TODO  Pack our own reply.
	// TODO  cache said reply.
	// TODO serve from cache (with fixed msg.Id) when possible.

	qname := r.Question[0].Name         // This is OUR name; so use it in our response
	ipString := w.RemoteAddr().String() // The user is from where?. dns.go only gives us strings.
	qtypeStr := "UNKNOWN"               // Default until we know better
	qnameLC := strings.ToLower(qname)   // We will ask for lowercase everything internally.

	view, _, _ := findView(ipString) // Geo + Resolver -> which data name in zone.conf

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		r.Question[0].Qtype == dns.TypeAXFR ||
		r.Question[0].Qtype == dns.TypeIXFR {
		m.Rcode = dns.RcodeRefused
		statsMsg(r)
		statsMsg(m)
		w.WriteMsg(m)
		return
	}

	// We'll need to pass the Qtype as a string to Lookup
	if cl1, ok := dns.TypeToString[r.Question[0].Qtype]; ok {
		qtypeStr = cl1
	}

	// _ = fmt.Sprintf("%v %v %v", view, asnString, ispString)

	// We know all we care to about the client.
	// We should now see what we know in our zone data.
	//stuff := Lookup(qname, view, qtype string)

	// TOOD handle QCLASS not being IN

	stuff := LookupFrontEnd(qnameLC, view, qtypeStr)

	for _, s := range stuff.Ans {
		rr, err := ourNewRR(s)
		if err == nil {
			m.Answer = append(m.Answer, rr)
		} else {
			log.Printf("Problems parsing '%s': %v\n", s, err)
		}
	}

	for _, s := range stuff.Auth {
		rr, err := ourNewRR(s)
		if err == nil {
			m.Ns = append(m.Ns, rr)
		} else {
			log.Printf("Problems parsing '%s': %v\n", s, err)
		}
	}
	for _, s := range stuff.Add {
		rr, err := ourNewRR(s)
		if err == nil {
			m.Extra = append(m.Extra, rr)
		} else {
			log.Printf("Problems parsing '%s': %v\n", s, err)
		}
	}
	m.Rcode = stuff.Rcode
	m.Authoritative = stuff.Aa

	// Finish up.
	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
}

func statsMsg(reply *dns.Msg) {
	isResponse := reply.Response
	qname := reply.Question[0].Name
	qnameLC := strings.ToLower(qname)

	RcodeStr := "UNKNOWN_RCODE"
	qtypeStr := "UNKNONN_QTYPE"

	// We'll need to pass the Qtype as a string to Lookup
	if cl1, ok := dns.TypeToString[reply.Question[0].Qtype]; ok {
		qtypeStr = cl1
	}
	if cl1, ok := dns.RcodeToString[reply.Rcode]; ok {
		RcodeStr = cl1
	}

	if isResponse {
		statsResponse.Increment(RcodeStr)

		/*
			// TODO Find a cheap way to figure out if our response is
			// a wildcard response, or a legit direct answer.
			// Why? Wildcards will polute the scoreboard - badly.
			// For now, skip storing names.
			if reply.Rcode == dns.RcodeSuccess {
				statsQname.Add(qnameLC, 1)
			}
		*/

		// If we needed the Vixie 0x20 bit hack for entropy,
		// make a note of it in the stats.  Might be useful.
		if qname != qnameLC {
			statsResponse.Increment("0x20")
		}
	} else {
		statsQuery.Increment(qtypeStr)

	}

}

func vixie0x20HackMsg(reply *dns.Msg) (changed bool) {
	// If qname is not entirely lowercase, then
	// spend extra cycles to modify all the names
	// to meet the 0x20 hack
	qname := reply.Question[0].Name
	qnameLC := strings.ToLower(qname)

	if qnameLC == qname {
		return changed // Do nothing.
	}

	// I want to try all three of r.Answer, r.Ns, and r.Extra
	// preferably without function call overhead.  This seems
	// awkward.
	try := append(reply.Answer, append(reply.Ns, reply.Extra...)...)
	for _, ptr := range try {
		// ptr  RR

		s := ptr.Header().Name
		if strings.HasSuffix(s, qnameLC) {

			keep := len(s) - len(qnameLC)
			s2 := s[0:keep] + qname
			ptr.Header().Name = s2 // Replace with new mixedcase name
			changed = true
		}
	}
	return changed
}
