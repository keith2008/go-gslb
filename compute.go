// compute will identify all the RRs for a given name, and return them
// in text form.
// This will not actually return DNS packets - just the DNS raw text
// needed to generate a reply.

package main

// INTERNAL NOTES

// HandleGSLB ->
//  LookupFrontEnd  ->
//   LookupFrontEndNoCache ->
//    LookupBackEnd

// func LookupFrontEnd(qname string, view string, qtype string) LookupResults
//	Calls LookupFrontEndNoCache if needed; uses cache if it can, rotates A/AAAA from the cache.
//	May cache - but only if there were results worth returning.  Otherwise, assumes it is a
//	garbage query, as there is typically no value to caching NXDOMAIN.

// func LookupFrontEndNoCache(qname string, view string, qtype string) LookupResults
//	Calls LookupBackEnd; identifies missing glue and handles DELEGATE.
//      Returns a finished LookupResults object.

// func LookupBackEnd(zoneRef *Config, qname string, view string, skipHC bool, recursion int) []string
//	Recursively digs for a given name, honoring the ISP "view", and handling
//	EXPAND, CNAME, HC, FB.   These results are cached heavily to cut down on the cost
//	of related queries.
//	ZoneRef in this context is a copy of GlobalZoneData(), copied safely just once per query,
//	and passed through all helper functions lock-free.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/miekg/dns"
)

// TheOneAndOnlyTTL is used as the TTL on items lacking an expressed on; and all generated records.
// TODO: Update parser to support TTLs in zone.conf
var TheOneAndOnlyTTL = 300

// LookupResults is a record containing the DNS strings to return for a given question,
// plus the response code and authority bit
type LookupResults struct {
	Ans   []string // DNS "Answers" section
	Auth  []string // DNS "Authority" section - who is responsible for this record?
	Add   []string // DNS "Additional" section - aka the glue
	Aa    bool     // DNS "I'm speaking authoritatively" bit, my Answers are legit (disabled when DELEGATE'ing to another DNS server)
	Rcode int      // DNS response code,  such as NOERROR or NXDOMAIN.  Numeric types are in the "dns" package.
}

// myNSRe is used to find NS targets in a string of text
var myNSRe = regexp.MustCompile(`\bNS\s+(\S+)`) // Used for finding NS to add glue

// Used when tracing lookups
type LookupTrace struct {
	recursion int
	trace     []string
}

// NewLookupTrace provides a new tracing object
func NewLookupTrace() *LookupTrace {
	n := new(LookupTrace)
	n.trace = make([]string, 0, 0)
	return n
}
func NewLookupTraceOff() *LookupTrace {
	n := new(LookupTrace)
	return n
}

// Add will add a line of trace information, indented based on current recursion tracker.
func (n *LookupTrace) Add(recursion int, s string) {
	if n.trace != nil {
		indentStr := indentSpaces(recursion) // For indenting debug output
		if strings.HasSuffix(s, "\n") {
			padded := indentStr + s
			n.trace = append(n.trace, padded)

		} else {
			padded := indentStr + s + "\n"
			n.trace = append(n.trace, padded)
		}
	}
}
func (n *LookupTrace) Addf(recursion int, format string, a ...interface{}) {
	if n.trace != nil {
		s := fmt.Sprintf(format, a...)
		n.Add(recursion, s)
	}
}

// LookupFrontEnd will return a set of results based on the asked name, the ISP name, the query class, and query type.
// If cached, we can expect to see the DNS "Answers" action to rotate every time this result is fetched (done by cache layer)
// Results are cached; don't modify the underlying store.
func LookupFrontEnd(qname string, view string, qtype string, recursion int, trace *LookupTrace) LookupResults {
	qname = toLower(qname)

	// Canonicalize query to not include the ".";
	// sometimes queries are internally generated and they
	// may or may not have ".".  The config file is missing
	// most of the "."'s ..'
	if strings.HasSuffix(qname, ".") {
		qname = qname[0 : len(qname)-1] // Strip the "." at the end
	}
	if trace.trace == nil { // Skip when tracing, otherwise try and read/return the cache
		cached, ok := getLookupFECache(qname, view, qtype)
		if ok {
			return cached
		}
	}
	ret := LookupFrontEndNoCache(qname, view, qtype, recursion+1, trace) // Results are final
	setLookupFECache(qname, view, qtype, ret)                            // Dump to cache
	return ret                                                           // And return
}

// LookupFrontEndNoCache takes a query for a given name, view, class, and qtype;
// and returns the final results to give to the DNS library.
// This does all the "cooking" of the results needed.
// Calls are made to LookupBackEnd, then analyized to see what
// changes to the packet have to be made.
//   NXDOMAIN vs empty NOERROR
//   Zone delegation to other DNS servers
//   DNS glue records for anything we know about
// Note that we do NOT handle dynamic queries like "ip.test-ipv6.com" here.
// Only cacheable entries go here.  Special queries will get hand crafted results.
func LookupFrontEndNoCache(qname string, view string, qtype string, recursion int, trace *LookupTrace) LookupResults {

	trace.Addf(0, "LookupFrontEndNoCache(%s,%s,%s)", qname, view, qtype)

	var results LookupResults
	results.Aa = true                // By default, be authoritive
	results.Rcode = dns.RcodeSuccess // NOERROR

	// We will need this possibly more than once.
	// Since there is potential lock contention, grab
	// it once - and then the recursions into LookupBackEnd
	// can avoid the mutex locks.
	zoneRef := GlobalZoneData() // Get the latest reference to the zone data

	// Go do a basic lookup.
	lookupList := LookupBackEnd(qname, view, false, zoneRef, recursion+1, trace)

	// We still have work to do.
	// We need to look for DELEGATE commands
	// We need to look for NS records that we can glue in
	for _, lookup := range lookupList {
		rtype := parseTokenFromString(lookup)
		data := CreateRRString(lookup, qname)

		// DELEGATE
		// When delegating, we will be rigid on output
		// No additionals other than the glue that DelegateNS
		// wants to add.
		if rtype == "DELEGATE" {
			return DelegateNS(zoneRef, qname, view, lookup, recursion+1, trace)
		}

		// CNAME send away immediately.
		// CNAME does not permit multiple RR types
		if rtype == "CNAME" {
			results.Ans = append(results.Ans, data)
			results.Auth = []string{} // Empty
			results.Add = []string{}  // Empty
			return results
		}

		// Do we want to include the current record?
		if qtype == "ANY" || rtype == qtype {
			results.Ans = append(results.Ans, data)
		}
	}

	if len(lookupList) == 0 { // No records at all.  So, REFUSED or NXDOMAIN ?
		return NotOurs(zoneRef, qname, view, recursion+1, trace) // REFUSED and NXDOMAIN both handled here
	}

	// No answers?  But we were authoritative?
	// That's bad luck.
	if len(results.Ans) == 0 {
		return NoAnswers(zoneRef, qname, view, recursion+1, trace)
	}

	// Yep, this is ours.  Add NS, possibly from a parent.
	if qtype != "NS" {
		trace.Addf(recursion, "Checking to see if we should add NS")
		nsname, ns := LookupWithParentsIfNeeded(zoneRef, qname, view, "NS", recursion+1, trace)
		for _, line := range ns {
			data := CreateRRString(line, nsname)      // SPECIFY the found NS name here - it miht be a parent
			results.Auth = append(results.Auth, data) // NS goes into the AUTH section when stapled with other results
		}
	}

	// Scan NS, identify any missing A/AAAA glue for the NS that we are auth for
	seencache := make(map[string]bool, 100)      // When we add glue, track records we've seen
	combined := []string{}                       // Start a list of answers we want to audit
	combined = append(combined, results.Auth...) // If we see any NS in AUTH
	combined = append(combined, results.Ans...)  // or even in ANS
	for _, line := range combined {              // For every NS and ANS in the combined list
		matches := myNSRe.FindStringSubmatch(line) // Check with a regex for the NS name
		if len(matches) > 0 {                      // If any NS was found
			ns := matches[1]                    // Grab the NS name from the regex capture
			if seen, _ := seencache[ns]; seen { // Have we already seen this NS?
				continue // We already saw it.
			}
			seencache[ns] = true // Note that we've seen it for next time.
			trace.Addf(recursion, "Found NS, checking for glue for %s", ns)

			possibleGlue := LookupBackEnd(ns, view, true, zoneRef, recursion+1, trace) // See what we know about that NS
			for _, possibleLine := range possibleGlue {                                // For each record in the lookup name
				r := parseTokenFromString(possibleLine) // Find out what RR type that record is
				if r == "A" || r == "AAAA" {            // If it is A or AAAA, we want it
					data := CreateRRString(possibleLine, ns) // to create glue
					results.Add = append(results.Add, data)  // to be stored into the Additional field
				}
			}
		}
	}
	return results
}

// NotOurs - used when we know nothing.
// May be NXDOMAIN or REFUSED, depending
func NotOurs(zoneRef *Config, qname string, view string, recursion int, trace *LookupTrace) LookupResults {
	trace.Addf(recursion, "NotOurs(%s,%s)", qname, view)

	var results LookupResults
	soaname, strList := LookupWithParentsIfNeeded(zoneRef, qname, view, "SOA", recursion+1, trace)
	if len(strList) == 0 {
		results.Aa = false               // This isn't our domain.
		results.Rcode = dns.RcodeRefused // REFUSED
	} else {
		results.Aa = true                  // This is our domain.  We own NXDOMAIN.
		results.Rcode = dns.RcodeNameError // NXDOMAIN
		for _, soa := range strList {
			data := CreateRRString(soa, soaname)
			results.Auth = append(results.Auth, data)
		}
	}
	return results
}

// NoAnswers - used when we do know the name, but
// don't have any records for the given type asked
func NoAnswers(zoneRef *Config, qname string, view string, recursion int, trace *LookupTrace) LookupResults {
	trace.Addf(recursion, "NoAnswers(%s,%s)", qname, view)

	var results LookupResults
	results.Aa = true                // We know this domain. We know it has no answers.
	results.Rcode = dns.RcodeSuccess // NOERROR

	soaname, strList := LookupWithParentsIfNeeded(zoneRef, qname, view, "SOA", recursion+1, trace)
	for _, soa := range strList {
		data := CreateRRString(soa, soaname)
		results.Auth = append(results.Auth, data)
	}
	return results
}

// DelegateNS will hand craft a response for a domain
// that has been DELEGATE'd to another location.
//  "I'm not the authority for this data; go elsewhere".
func DelegateNS(zoneRef *Config, qname string, view string, delegate string, recursion int, trace *LookupTrace) LookupResults {
	if trace != nil {
		trace.Addf(2, "DelegateNS(%s,%s,%s)", qname, view, delegate)
	}

	var results LookupResults
	results.Aa = false // never authoritive when delegating away
	words := QuotedStringToWords(delegate)

	if len(words) >= 3 {
		_, from, toList := words[0], words[1], words[2:]
		for _, to := range toList {

			// Add in the NS to AUTH
			s := fmt.Sprintf("%s. %v NS %s", from, TheOneAndOnlyTTL, to)
			results.Auth = append(results.Add, s)

			// Add in the glue for additional
			ipList := LookupBackEnd(to, view, false, zoneRef, recursion+1, trace)
			for _, record := range ipList {
				r := parseTokenFromString(record)
				if r == "A" || r == "AAAA" {
					d := CreateRRString(record, to)
					results.Add = append(results.Add, d)
				}
			}

		}
	}
	return results
}

// LookupWithParentsIfNeeded - given a name, a view, *and* a RR type
// will find the records for the name (or a parent name) with the matching RR
// Mainly used for building NS and SOA records
// TODO: Announce a countest for a better function name to replace "LookupWithParentsIfNeeded"
func LookupWithParentsIfNeeded(zoneRef *Config, qname string, view string, token string, recursion int, trace *LookupTrace) (record string, lines []string) {
	if trace != nil {
		trace.Addf(recursion, "LookupWithParentsIfNeeded(%s,%s,%s)", qname, view, token)
	}

	name := qname
	matches := []string{}
	for strings.Contains(name, ".") {
		lookup := LookupBackEnd(name, view, true, zoneRef, recursion+1, trace) // Do we know anything about this name?
		for _, line := range lookup {
			t := parseTokenFromString(line)
			if t == token {
				matches = append(matches, line)
			}
		}
		if len(matches) > 0 {
			return name, matches
		}
		sp := strings.SplitN(name, ".", 2) // Split on first "."
		name = sp[1]                       // And strip the first name
	}
	return "", matches
}

// parseTokenFromString - Given "A 192.0.2.1", returns simply "A"
func parseTokenFromString(line string) (rtype string) {
	words := QuotedStringToWords(line)
	//words := strings.SplitN(line, " ", 2)
	if len(words) > 0 {
		return toUpper(words[0])
	}

	return ""

}

// CreateRRString - Given data from zone.conf, returns
// a parsed (by words) set of strings.  The first word will be
// made all-caps, as that represents the RR type.
// Quoted strings are preserved as single tokens.
// Finally, since our zone data presumes that our input is
// without trailing dots, this function will fix the trailing dots
// both for the rname as well as the target of CNAME, NS, MX, and SRV.
func CreateRRString(line string, resourceName string) (record string) {

	// Break into shell words.
	// Sort of.  Observation: any "words" that were
	// quoted, still have quotes!
	words := QuotedStringToWords(line)

	if len(words) > 1 {
		rtype := toUpper(words[0])
		remainder := []string{}
		for _, s := range words[1:] {
			if rtype == "TXT" || rtype == "SPF" || strings.ContainsAny(s, " \t") {
				//Bring this back if the earlier quotes disappear
				//TODO:   QuotedStringToWords needs to not pass quotes.
				//s = fmt.Sprintf("\"%s\"", s) // Quote TXT, SPF, and anthing with whitespace
			} else {
				if s == words[len(words)-1] {
					s = toLower(s) // Canonicalize target names as lower case
				}
			}
			remainder = append(remainder, s)
		}

		// Does the first word (the name) end in a dot?  If not, fix it.
		if strings.HasSuffix(resourceName, ".") == false {
			resourceName = resourceName + "."
		}

		// Depending on what the RTYPE is, we might take some liberties
		// and force canonicalize on input the target of CNAME, NS, MX, and SRV.
		// Better to do this at generation time, instead of per-query.
		var data string
		switch rtype {
		case "CNAME", "NS", "MX", "SRV":
			data = fmt.Sprintf("%s %v %s %s", resourceName, TheOneAndOnlyTTL, rtype, toLower(strings.Join(remainder, " ")))

		default:
			data = fmt.Sprintf("%s %v %s %s", resourceName, TheOneAndOnlyTTL, rtype, strings.Join(remainder, " "))
		}

		// The shitty thing about keeping all this stuff in plain human readable
		// is that the trailing dots are needed but often missed.
		switch rtype {
		case "CNAME", "NS", "MX", "SRV":
			if strings.HasSuffix(data, ".") == false {
				data = data + "."
			}
		}
		return data
	}
	return line
}

// LookupBackEnd will take just the qname and view, and return all records (as strings)
// without regard as to token type.  EXPAND CNAME HC and FB are expanded.
// No glue work is done; no evaluating the results is done.  Just simple expansion
// with health checks factored in.
func LookupBackEnd(qname string, view string, skipHC bool, zoneRef *Config, recursion int, trace *LookupTrace) []string {

	if trace != nil {
		trace.Addf(recursion, "LookupBackEnd(%s,%s,%v)", qname, view, skipHC)
	}

	// Strip trailing "." if found
	if strings.HasSuffix(qname, ".") {
		qname = qname[0 : len(qname)-1]
	}

	// Check the cache. If found, return the cached values.
	if cached, ok := getLookupBECache(qname, view, skipHC); ok {
		return cached
	}

	returnData := []string{} // Container to return results to the caller

	name := toLower(qname) // Make lower case.

	found, ok := zoneRef.GetSectionNameValueStrings(view, name) // Find the view-specific (or default) strings for the name

	if (ok) && (len(found) > 0) {
		hcFound := false // Keep track of whether any HC (Health Check) lines were seen

	loop:
		for _, line := range found {
			words := QuotedStringToWords(line) // Tokenize for processing
			token := toUpper(words[0])         // Simplifies checking if we only look at all-caps

			// Health checks. If the HC is good, translate into an EXPAND.
			// If the HC is bad, then simply skip the line.
			// If skip_hc is set, then we ignore the health check entirely.
			if token == "HC" {
				if len(words) >= 3 {
					hcFound = true
					hc := words[1]
					target := words[2]
					keep, _ := GetStatus(hc, target)
					if trace != nil {
						trace.Add(recursion, fmt.Sprintf("HC %s %s %v", hc, target, keep))
					}
					if keep || skipHC {
						words = []string{"EXPAND", target}
						token = "EXPAND"
						// We will continue processing this line, don't exit early.
					} else {
						continue loop // Skip this line.  It is dead to us.
					}
				}
			}

			// If the token is "FB", we only want to process this line
			// if we have no other A/AAAA records.
			if token == "FB" {
				// Check "ret" for A/AAAA
				// Figure out how to do this ASAP
				for _, v := range returnData {
					w := QuotedStringToWords(v)
					if len(w) >= 1 {
						if w[0] == "A" || w[0] == "AAAA" {
							continue loop // No FB needed
						}
					}
				}
				token = "EXPAND" // Convert to EXPAND, we do need this fallback
				trace.Add(recursion, "FB needed")
			}

			// Expand and CNAME will recursively pull in other strings.
			if token == "EXPAND" || token == "CNAME" || token == "FB" {
				if len(words) >= 2 {
					try := words[1]

					trace.Addf(recursion, "%s %s", words[0], words[1])

					more := LookupBackEnd(try, view, skipHC, zoneRef, recursion+1, trace)

					if len(more) > 0 {
						// CNAME, if found locally, will be treated like EXPAND to save a round-trip to the DNS server.
						returnData = append(returnData, more...)

					} else {
						// Not found?
						if token == "CNAME" {
							returnData = append(returnData, line) // Keep the CNAME as-is
						} else {
							Debugf("LookupBackEnd: %v asked to %v %v; not found\n", name, token, try)
						}
					}
				}
				// In all cases, CNAME and EXPAND, we will have done everything we want to
				// for this line; and want to not do anything else.
				continue loop
			}

			// Everything else? Just pass it.
			returnData = append(returnData, line)

		}

		// Hey, did we see any HC lines?  If so, make sure we have at least one A or AAAA line.
		// This could be a bit cheaper if we tracked this better while building,
		// but I Actually wantt o check for A/AAAA *after* any real or virtual EXPAND statements.
		// Only one way to do that...

		if (hcFound == true) && (skipHC == false) {
			needRerun := true

			for _, v := range returnData {
				w := QuotedStringToWords(v)
				if len(w) >= 1 {
					if w[0] == "A" || w[0] == "AAAA" {
						needRerun = false
						break
					}
				}
			}

			if needRerun {
				trace.Add(recursion, "LookupBackEnd: Rerunning with health checks disabled")
				returnData = LookupBackEnd(name, view, true, zoneRef, recursion+1, trace)
			}
		}

	} else {
		// Try wildcards?
		if len(name) > 2 && name[0:2] != "*." { // What about the wildcard?
			sp := strings.SplitN(name, ".", 2) // Split the name into the first hostname, and the remainder
			if len(sp) > 1 {
				try := "*." + sp[1] // Replace the hostname with a *, only if we found a "."
				returnData = LookupBackEnd(try, view, skipHC, zoneRef, recursion+1, trace)
			}
		}
	}

	// Cache the results, but only if non-empty.
	// Cache writes are too expensive to waste on empty results at this layer;
	// we can count on the front end layer to cache repeated queries to the same
	// name.
	if len(returnData) > 0 {
		setLookupBECache(qname, view, skipHC, returnData)
	}
	return returnData
}
