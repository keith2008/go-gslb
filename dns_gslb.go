package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"

	"github.com/miekg/dns"
)

// NOTRACE is a shared trace config (basically: "no trace") that has no need to share the audit trail.
// No trail = shared across all instances without regards to thread safety;
// and just slightly less malloc'ing on function invocation.
var NOTRACE = NewLookupTraceOff()

// findViewOnly will cache.
func findViewOnly(ipString string) (view string) {
	ip, _, err := net.SplitHostPort(ipString)
	if err == nil {
		ipString = ip // With the :portnumber removed.
	}
	if val, ok := CacheView.Get(ipString); ok {
		return val
	}
	view, _, _ = findView(ipString) // view,asn,ispname
	if view != "" {
		CacheView.Set(ipString, view)
	}
	return view
}

// findView will (for a given IP string) return the "view" (ie, "comcast" or "default",
// the asn number (as a string), and the ISP info (as a string).
// This is not cached.
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
	if found, ok := CacheRR.Get(s); ok {
		deep := dns.Copy(found)
		return deep, nil

	}
	// Even a fresh instance will be deep copied.
	parsed, err := dns.NewRR(s)
	if err == nil {
		CacheRR.Set(s, parsed)
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
	qtype := r.Question[0].Qtype        // What are we asking for?
	qtypeStr := qtypeToString(qtype)    // What are we asking for? A STRING!
	qnameLC := toLower(qname)           // We will ask for lowercase everything internally.
	wasLC := qname == qnameLC           // We really care about the case that people us when asking.
	view := findViewOnly(ipString)      // Geo + Resolver -> which data name in zone.conf

	QI := QueryInfo{qname: qname, view: view, qtype: qtypeStr}

	// Hey.  Maybe we can return cached data?
	if wasLC {
		if cached, ok := CacheMsgs.Get(QI); ok {
			j := rand.Intn(len(cached))   // We expect multiple possible results; answer one at random.
			bits := int(cached[j].msg[2]) // We need to figure out how to set/clear the RD bit
			if r.RecursionDesired {
				bits = bits | 0x01 // Set RD
			} else {
				bits = bits &^ 0x01 // Clear RD
			}

			// Build new data packet, with the new 3 bytes based on the current
			// caller; and the remaining bytes on the cached data.
			newLeader := []byte{uint8(r.Id >> 8), uint8(r.Id & 0xff), uint8(bits)}
			newData := append(newLeader, cached[j].msg[3:]...)
			w.Write(newData)

			// Don't forget the stats.
			statsQuery.Increment(qtypeStr)
			statsResponse.Increment(cached[j].rcodeStr)

			return
		}
	}

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

	// Go do real computational work to see what our records should say.
	// stuff := LookupFrontEnd(qnameLC, view, qtypeStr, 0, NOTRACE)
	stuff := LookupFrontEndNoCache(qnameLC, view, qtypeStr, 0, NOTRACE)

	// Shuffle, to randomize answers, if we got more than one.
	if len(stuff.Ans) > 1 {
		n := len(stuff.Ans)
		for i := n - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			stuff.Ans[i], stuff.Ans[j] = stuff.Ans[j], stuff.Ans[i]
		}
	}

	// Copy the results to fully formed RRs and stuff them into our
	// message.
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

	if len(stuff.Ans) > 1 {
		n := len(stuff.Ans)
		for i := n - 1; i > 0; i-- {
			j := rand.Intn(i + 1)
			stuff.Ans[i], stuff.Ans[j] = stuff.Ans[j], stuff.Ans[i]
		}
	}

	// Finish up.
	if wasLC {
		vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	}

	// Save some stats
	statsMsg(r)
	statsMsg(m)

	// Finally, pack, possibly cache, and write the dns response
	data, err := m.Pack()
	if err != nil {
		// We had an error creating a DNS packet?
		log.Printf("Error with m.Pack %v", err)
		return
	}

	if wasLC == true {
		// Hey, we can cache this.
		// No MixEdCaSE
		rcodeStr := rcodeToString(stuff.Rcode) // For stats

		group := []MsgCacheRecord{} // Allocate a new set of pointers
		group = append(group, freshMsgCacheRecord(data, rcodeStr))

		// Calculate the remaining rotations
		for i := 1; i < len(stuff.Ans); i++ { // We already did "0"
			m.Answer = append(m.Answer[1:], m.Answer[0]) // One DNS RR rotation
			data, err = m.Pack()                         // Re-pack the DNS data
			if err == nil {                              // If no error..
				group = append(group, freshMsgCacheRecord(data, rcodeStr))
			}
		}
		CacheMsgs.Set(QI, group)
	}

	w.Write(data)
}

func dupedata(b []byte) []byte {
	n := make([]byte, len(b))
	copy(n, b)
	return n
}
func freshMsgCacheRecord(data []byte, rcodeStr string) (m MsgCacheRecord) {
	m.msg = dupedata(data)
	m.rcodeStr = rcodeStr
	return m
}

func rcodeToString(rcode int) string {
	if cl1, ok := dns.RcodeToString[rcode]; ok {
		return cl1
	}
	return fmt.Sprintf("Rcode%v", rcode)

}
func qtypeToString(qtype uint16) string {
	if cl1, ok := dns.TypeToString[qtype]; ok {
		return cl1
	}
	return fmt.Sprintf("Qtype%v", qtype)
}

// WebHandleTrace serves /gslb/trace/HOSTNAME
func WebHandleTrace(w http.ResponseWriter, r *http.Request) {
	trace := NewLookupTrace()
	myHTTPGslbTrace(w, r, trace)

}

// WebHandleLookup serves /gslb/lookup/HOSTNAME
func WebHandleLookup(w http.ResponseWriter, r *http.Request) {
	notrace := NewLookupTraceOff()
	myHTTPGslbTrace(w, r, notrace)
}

// handleReflectIP responds with the caller's IP address,
// in the form of A/AAAA as well as TXT
func myHTTPGslbTrace(w http.ResponseWriter, r *http.Request, trace *LookupTrace) {

	qname := "unspecified"
	qtypeStr := "A"
	view := "default"

	//   /gslb/trace/test-ipv6.com
	//   /gslb/trace/test-ipv6.com/A
	//   /gslb/trace/test-ipv6.com/A/comcast
	words := strings.Split(r.RequestURI, "/")
	if len(words) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, word := range words[3:] {
		uc := toUpper(word)
		lc := toLower(word)

		// Easy one first.  DNS types
		if _, ok := dns.StringToType[uc]; ok {
			qtypeStr = uc
			continue
		}

		// "Views" - by IP address or AS number.
		I := GlobalViewData()
		if found, ok := I.GetSectionNameValueString("default", word); ok {
			view = found
			continue
		}
		if found, ok := I.GetSectionNameValueString("default", lc); ok {
			view = found
			continue
		}

		// Otherwise, is it a hostname or a view (by name)?
		// Assumption: views don't use dots.
		if strings.ContainsAny(word, ".") {
			qname = word

		} else {
			view = word
		}
	}

	qnameLC := toLower(qname)
	trace.Addf(0, "Looking up qname=%s qtype=%s view=%s", qnameLC, qtypeStr, view)
	trace.Addf(0, "")

	stuff := LookupFrontEnd(qnameLC, view, qtypeStr, 0, trace)

	w.Header().Set("Content-Type", "text/plain")
	text := strings.Join(trace.trace, "")
	io.WriteString(w, text)
	io.WriteString(w, "\n")
	io.WriteString(w, fmt.Sprintf("QNAME: %v\n", qnameLC))

	io.WriteString(w, fmt.Sprintf("RCODE: %v AA: %v\n", rcodeToString(stuff.Rcode), stuff.Aa))
	io.WriteString(w, "\n")

	if len(stuff.Ans) > 0 {
		io.WriteString(w, "Answers:\n")
		for _, s := range stuff.Ans {
			io.WriteString(w, "  "+s+"\n")
		}
	}

	if len(stuff.Auth) > 0 {
		io.WriteString(w, "Auth:\n")
		for _, s := range stuff.Auth {
			io.WriteString(w, "  "+s+"\n")
		}
	}

	if len(stuff.Add) > 0 {
		io.WriteString(w, "Additional:\n")
		for _, s := range stuff.Add {
			io.WriteString(w, "  "+s+"\n")
		}
	}

}

func statsMsg(reply *dns.Msg) {
	isResponse := reply.Response
	qname := reply.Question[0].Name
	qnameLC := toLower(qname)

	RcodeStr := rcodeToString(reply.Rcode)
	qtypeStr := qtypeToString(reply.Question[0].Qtype)

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
	qnameLC := toLower(qname)

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

func init() {
	http.HandleFunc("/gslb/trace/", WebHandleTrace)
	http.HandleFunc("/gslb/lookup/", WebHandleLookup)

}
