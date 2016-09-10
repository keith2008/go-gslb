package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/miekg/dns"
)

// handleAS replies with TXT "as=1234"
func handleAS(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeTXT && r.Question[0].Qtype != dns.TypeANY) {
		statsMsg(r)
		statsMsg(m)
		w.WriteMsg(m)
		return
	}

	qname := r.Question[0].Name // This is OUR name; so use it in our response

	list := []string{}
	list = append(list, w.RemoteAddr().String())
	ipString, subnetString, subnetSpecified, newSubnetOpt := getClientInfo(w, r)

	_ = subnetString
	if subnetSpecified {
		list = append(list, ipString)
		m.Extra = append(m.Extra, newSubnetOpt)
	}

	for _, ipString = range list {
		log.Printf("ipstring is now %s", ipString)
		_, asnString, _, _ := findView(ipString) // Geo + Resolver -> which data name in zone.conf
		//view, asnString, ispString := findView(ipString) // Geo + Resolver -> which data name in zone.conf
		txt := fmt.Sprintf("ip=%s as='%v'", myQuote(parseIpOnly(ipString)), asnString)

		rr, err := ourNewRR(fmt.Sprintf("%s 0 TXT %s", qname, `"`+txt+`"`))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}

	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
	return
}

// handleHelp replies with TXT records indicating all known names
func handleHelp(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeTXT && r.Question[0].Qtype != dns.TypeANY) {
		w.WriteMsg(m)
		return
	}
	qname := r.Question[0].Name // This is OUR name; so use it in our response

	c := GlobalConfig() // Get our config object

	handleHelpHelper := func(qname, name string, extratext string) (ret []dns.RR) {
		s, _ := c.GetSectionNameValueStrings("special", name)

		if s != nil {
			for _, dom := range s {
				pattern := dom
				if !(strings.HasSuffix(pattern, ".")) {
					pattern = pattern + "."
				}
				txt := fmt.Sprintf(`%s 0 TXT "%s %s"`, qname, pattern, extratext)
				rr, err := ourNewRR(txt)
				if err == nil {
					ret = append(ret, rr)
				}
			}
		}
		return ret
	}

	for _, rr := range handleHelpHelper(qname, "ip", "Reports your IP information") {
		m.Answer = append(m.Answer, rr)
	}
	for _, rr := range handleHelpHelper(qname, "as", "Reports your ISP's BGP ASN") {
		m.Answer = append(m.Answer, rr)
	}
	for _, rr := range handleHelpHelper(qname, "provider", "Reports your ISP's name") {
		m.Answer = append(m.Answer, rr)
	}
	for _, rr := range handleHelpHelper(qname, "country", "Reports your ISP's country") {
		m.Answer = append(m.Answer, rr)
	}
	for _, rr := range handleHelpHelper(qname, "maxmind", "Reports what we know from MaxMind") {
		m.Answer = append(m.Answer, rr)
	}

	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
	return
}

// handleISP replies with TXT "isp=Provider Name"
func handleISP(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeTXT && r.Question[0].Qtype != dns.TypeANY) {
		w.WriteMsg(m)
		return
	}

	qname := r.Question[0].Name // This is OUR name; so use it in our response

	list := []string{}
	list = append(list, w.RemoteAddr().String())
	ipString, subnetString, subnetSpecified, newSubnetOpt := getClientInfo(w, r)
	_ = subnetString
	if subnetSpecified {
		list = append(list, ipString)
		m.Extra = append(m.Extra, newSubnetOpt)
	}
	for _, ipString = range list {
		_, _, txt, _ := findView(ipString) // Geo + Resolver -> which data name in zone.conf
		txt = fmt.Sprintf("ip=%s isp=%s", myQuote(ipString), myQuote(txt))
		rr, err := ourNewRR(fmt.Sprintf("%s 0 TXT %s", qname, `"`+txt+`"`))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}

	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
	return
}

// handleCountry replies with TXT "country=US"
func handleCountry(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeTXT && r.Question[0].Qtype != dns.TypeANY) {
		w.WriteMsg(m)
		return
	}

	qname := r.Question[0].Name // This is OUR name; so use it in our response

	list := []string{}
	list = append(list, w.RemoteAddr().String())
	ipString, subnetString, subnetSpecified, newSubnetOpt := getClientInfo(w, r)
	_ = subnetString
	if subnetSpecified {
		list = append(list, ipString)
		m.Extra = append(m.Extra, newSubnetOpt)
	}

	for _, ipString = range list {
		_, _, _, country := findView(ipString) // Geo + Resolver -> which data name in zone.conf
		txt := fmt.Sprintf("ip=%s country=%s", myQuote(ipString), myQuote(country))
		rr, err := ourNewRR(fmt.Sprintf("%s 0 TXT %s", qname, `"`+txt+`"`))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}
	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
	return
}

func parseIpOnly(s string) string {
	if a, _, err := net.SplitHostPort(s); err == nil {
		return a
	}
	return s
}

// handleMaxMind replies with ip=74.125.187.158 as=15169 isp='Google Inc.'"
func handleMaxMind(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true
	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeTXT && r.Question[0].Qtype != dns.TypeANY) {
		w.WriteMsg(m)
		return
	}

	qname := r.Question[0].Name // This is OUR name; so use it in our response

	list := []string{}
	list = append(list, w.RemoteAddr().String())
	ipString, subnetString, subnetSpecified, newSubnetOpt := getClientInfo(w, r)

	_ = subnetString
	if subnetSpecified {
		list = append(list, ipString)
		m.Extra = append(m.Extra, newSubnetOpt)
	}
	for _, ipString = range list {
		log.Printf("ipString is now %s\n", ipString)
		_, asn, txt, country := findView(ipString) // Geo + Resolver -> which data name in zone.conf
		txt = fmt.Sprintf("ip=%s as='%v' isp=%s country=%s", myQuote(parseIpOnly(ipString)), asn, myQuote(txt), myQuote(country))
		rr, err := ourNewRR(fmt.Sprintf("%s 0 TXT %s", qname, `"`+txt+`"`))
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}

	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
	return
}

// handleView replies with TXT view=default
func handleView(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	// Reasons to refuse to answer, there are many.
	if r.Question[0].Qclass != dns.ClassINET ||
		(r.Question[0].Qtype != dns.TypeTXT && r.Question[0].Qtype != dns.TypeANY) {
		w.WriteMsg(m)
		return
	}

	qname := r.Question[0].Name // This is OUR name; so use it in our response

	ipString, subnetString, subnetSpecified, newSubnetOpt := getClientInfo(w, r)
	_ = subnetString
	if subnetSpecified {
		m.Extra = append(m.Extra, newSubnetOpt)
	}

	view, _, _, _ := findView(ipString) // Geo + Resolver -> which data name in zone.conf
	txt := fmt.Sprintf("ip=%s view=%s", myQuote(ipString), myQuote(view))
	rr, err := ourNewRR(fmt.Sprintf("%s 0 TXT %s", qname, `"`+txt+`"`))
	if err == nil {
		m.Answer = append(m.Answer, rr)
	}

	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
	return
}

func myQuote(s string) string {
	s = strings.Replace(s, `"`, `\"`, -1)
	s = strings.Replace(s, `'`, `\'`, -1)
	s = `'` + s + `'`
	return s
}

// handleIP responds with the caller's IP address,
// in the form of A/AAAA as well as TXT.
// TXT will indicate the source address, port number,
// and whether it was UDP or TCP.
func handleIP(w dns.ResponseWriter, r *dns.Msg) {
	// handleReflectIP is from github.com/miekg/exdns/reflect
	// originally written Miek Gieben <miek@miek.nl>
	// modified for my own tastes here. <jfesler@gigo.com>

	var (
		v4  bool
		rr  dns.RR
		str string
		a   net.IP
	)
	qname := r.Question[0].Name // This is OUR name; so use it in our response

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	switch r.Question[0].Qtype {

	case dns.TypeAXFR, dns.TypeIXFR:
		m.SetRcode(r, dns.RcodeRefused) // Actively refuse.

	case dns.TypeA, dns.TypeAAAA, dns.TypeANY, dns.TypeTXT:

		// Only do real work fo-r A, AAAA, and TXT requests.

		if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
			str = fmt.Sprintf("ip=%s protocol=%s", myQuote(ip.String()), myQuote("udp"))
			a = ip.IP
			v4 = a.To4() != nil
		}
		if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
			str = fmt.Sprintf("ip=%s protocol=%s", myQuote(ip.String()), myQuote("tcp"))
			a = ip.IP
			v4 = a.To4() != nil
		}

		if v4 == true {
			if r.Question[0].Qtype == dns.TypeA || r.Question[0].Qtype == dns.TypeANY {
				rr = new(dns.A)
				rr.(*dns.A).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}
				rr.(*dns.A).A = a.To4()
				m.Answer = append(m.Answer, rr)
			}
		}
		if v4 == false {
			if r.Question[0].Qtype == dns.TypeAAAA || r.Question[0].Qtype == dns.TypeANY {
				rr = new(dns.AAAA)
				rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0}
				rr.(*dns.AAAA).AAAA = a
				m.Answer = append(m.Answer, rr)
			}
		}

		t := new(dns.TXT)
		t.Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0}
		t.Txt = []string{str}
		if r.Question[0].Qtype == dns.TypeANY || r.Question[0].Qtype == dns.TypeTXT {
			m.Answer = append(m.Answer, t)
		} else {
			m.Extra = append(m.Extra, t)
		}

		ipString, subnetString, subnetSpecified, newSubnetOpt := getClientInfo(w, r)
		if subnetSpecified {
			str = fmt.Sprintf("ip=%s protocol=%s", myQuote(subnetString), myQuote("edns0-client-subnet"))
			a = net.ParseIP(ipString)
			v4 = a.To4() != nil

			if v4 == true {
				if r.Question[0].Qtype == dns.TypeA || r.Question[0].Qtype == dns.TypeANY {
					rr = new(dns.A)
					rr.(*dns.A).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}
					rr.(*dns.A).A = a.To4()
					m.Answer = append(m.Answer, rr)
				}
			}
			if v4 == false {
				if r.Question[0].Qtype == dns.TypeAAAA || r.Question[0].Qtype == dns.TypeANY {
					rr = new(dns.AAAA)
					rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0}
					rr.(*dns.AAAA).AAAA = a
					m.Answer = append(m.Answer, rr)
				}
			}

			t = new(dns.TXT)
			t.Hdr = dns.RR_Header{Name: qname, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0}
			t.Txt = []string{str}
			if r.Question[0].Qtype == dns.TypeANY || r.Question[0].Qtype == dns.TypeTXT {
				m.Answer = append(m.Answer, t)
			} else {
				m.Extra = append(m.Extra, t)
			}
			m.Extra = append(m.Extra, newSubnetOpt)
		}

	}
	// Finish up.
	vixie0x20HackMsg(m) // Handle MixEdCase.org requests
	statsMsg(r)
	statsMsg(m)
	w.WriteMsg(m)
}
