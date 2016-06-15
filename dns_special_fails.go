package main

import (
	"strings"
	"regexp"
	"github.com/miekg/dns"
)



// Handle magic names like:

// a-good-aaaa-good.dns-test.net
// a-timeout-aaaa-good.dns-test.net
// a-refused-aaaa-good.dns-test.net
// a-servfail-aaaa-good.dns-test.net
// a-good-aaaa-timeout.dns-test.net
// a-good-aaaa-refused.dns-test.net
// a-good-aaaa-servfail.dns-test.net



func handleBreak(w dns.ResponseWriter, r *dns.Msg) {
	        var re *regexp.Regexp

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	if r.Question[0].Qclass == dns.ClassINET {
	  qname := r.Question[0].Name
	  qname = strings.ToLower(qname)
	  if r.Question[0].Qtype == dns.TypeA || r.Question[0].Qtype == dns.TypeANY {
	        re = regexp.MustCompile(`\ba-timeout\b`)
	        if re.MatchString(qname) {
	  		return // DO NOTHING but time out
	  	}
	        re = regexp.MustCompile(`\ba-servfail\b`)
                if re.MatchString(qname) {
	  		m.SetRcode(r, dns.RcodeServerFailure) 
	  		statsMsg(r)
	                statsMsg(m)
	                w.WriteMsg(m)
        	        return
	  	}
	        re = regexp.MustCompile(`\ba-refused\b`)
                if re.MatchString(qname) {
	  		m.SetRcode(r, dns.RcodeRefused) 
	  		statsMsg(r)
	                statsMsg(m)
	                w.WriteMsg(m)
        	        return
	  	}
	  }
	  if r.Question[0].Qtype == dns.TypeAAAA || r.Question[0].Qtype == dns.TypeANY {
	        re = regexp.MustCompile(`\baaaa-timeout\b`)
	        if re.MatchString(qname) {
	  		return // DO NOTHING but time out
	  	}
	        re = regexp.MustCompile(`\baaaa-servfail\b`)
	        if re.MatchString(qname) {
	  		m.SetRcode(r, dns.RcodeServerFailure) 
	  		statsMsg(r)
	                statsMsg(m)
	                w.WriteMsg(m)
        	        return
	  	}
	        re = regexp.MustCompile(`\baaaa-refused\b`)
	        if re.MatchString(qname) {
	  		m.SetRcode(r, dns.RcodeRefused) 
	  		statsMsg(r)
	                statsMsg(m)
	                w.WriteMsg(m)
        	        return
	  	}
	  }
	}
	
	// Oterwise, call the original handler!
	handleGSLB(w,r)
	return
	
}
