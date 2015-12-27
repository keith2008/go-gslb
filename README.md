# Poor-man's GSLB

This is a homebrew GSLB (Global Server Load Balancer).  It uses DNS to route customers to locations based on a combination of their ISP, and the health check status of your sites.

DNS lookups are one way of routing customers to an available resource.   Decisions can be made that focus on proximity to the user (in our case, "isp" match); 

This is specifically used in the [falling-sky project](http://falling-sky.github.com); aka [test-ipv6.com](http://test-ipv6.com).

## What is a Global Service Load Balancer?

A Global Service Load Balancer (GSLB) uses the DNS system to route customers to the "best" available location.  While it may be slow to handle host-level failover (local load balancers such as HA-Proxy are better suited), it *is* good for handling the case of datacenter or cloud outages, to send customers to another location.

Commercial GSLB systems can use many inputs for deciding where to send customers.  Up/Down status are the most obvious; but they often also take into account the user's distance; the cost of providing services; or even political/legal considerations.

## Features

The feature set of this particular GSLB implementation is fairly small:

 * Per-ISP views based on ASN can override any host data.  
 * Fallback to DEFAULT values make ISP overrides simple and small
 * Health checking (pass/fail only).  
   * check_http verifies a valid HTTP response
   * check_irc simply checks port 6667
   * check_mirror verifies that a site is ready to be a falling-sky [transparent mirror](https://github.com/falling-sky/source/wiki/TransparentMirrors)
   * Other checks can be implmented as go code but must be compiled in.  All checks are internal for performance.
 * CNAME expansion - serves A/AAAA records immediately instead of the underlying CNAME, when the data is local
 * Simplified zone data format.
 * [0x20 bit hack](https://tools.ietf.org/html/draft-vixie-dnsext-dns0x20-00) provides additional entropy data for clients who request it.
 
 
### Missing Stuff
 
These are not part of the product and worth mentioning in the spirit of full disclosure.

 * DNSSEC is not yet implemented.  I'm not sure when I will do this.  The intent will be to do online signing on the fly, "some day".  
 * EDNS0 packet size.  My expected responses are all <512b.
 * ENDS0 client subnet.  This may get added sooner rather than later.
 * 
  


 
 

## Dependencies

 * GO 1.5, with `GOPATH` properly set up.
 * Miek Gieben's [github.com/miekg/dns](http://github.com/miekg/dns) library
 * MaxMind's [GeoLite CSV data](http://dev.maxmind.com/geoip/legacy/geolite/) for IPv4 and IPv6 ASN lookups.  
 

"This product includes GeoLite data created by MaxMind, available from [http://www.maxmind.com](http://www.maxmind.com)."

Tested on Ubuntu 14.04 and Mac OS X 10.11.  Any platform supported by Go should work.

## Configuration

### Real World configs

You can see real world configs here: 

https://github.com/falling-sky/gslb 


### server.conf 

The examples below are just that - and not actually tracking the real world examples any more.  They are meant to illustrate key points of how the configuration works.

The config format is a bastardization between INI and YAML.  At some point this may go back to pure YAML, but that depends on my figuring out how to handle unstructured data in Go.

```INI
[default]
maxcache: 100000

[server/hostname]
udp: [::]:53
tcp: [::]:53

[server/my-Macbook.local]
udp: 127.0.0.1:8053
tcp: 127.0.0.1:8053
udp: [::1]:8053
tcp: [::1]:8053

[interval]
check_mirror: 45
check_irc: 30
check_http: 30

[special]    
# Special handlers for these.
# Try dig txt maxmind.test-ipv6.com  for an example.
# Some of these handlers have multiple names.
ip: [ip.test-ipv6.com, what.test-ipv6.com]
as: [as.test-ipv6.com, asn.test-ipv6.com]
view: [which.test-ipv6.com, view.test-ipv6.com]
isp: provider.test-ipv6.com
maxmind: maxmind.test-ipv6.com
help: help.test-ipv6.com

```

Lookups against the configuration are first done against a specific section (based on the code; and if not found there, in `[default]`).

Inside each section is a series of one or more key: value pairs.

```INI
[section]
key: value1
key: value2
```

Because this is YAML inspired, there are other ways to indicate multiple values; use whatever gives you the best readability.

```INI
[section]
# Using YAML list syntax - note space after comma
example: [value1, value2, value3]

# Using YAML list syntax - longer form
example2:
  - value1
  - value2
```
  
Last, you can designate some sections of code to only take effect when you're running on a specific host (as designed by the `hostname` command):

```INI
[section/hostname1]
key: value for hostname1
[section/hostname2]
key: value for hostname2
```

*Caution* If you have multiple values listed like this, they will be parsed as multiple values (instead of one value):

```INI
[section]
key: value1
[section/hostname1]
key: value2   # WHoops
```

Because of this, it is recommended that any config variables you wish to have hostname specific overrides, should have no default.  Except, perhaps, in `[default]`.


### zone.conf

For a full example, see the bundled [etc/zone.conf](etc/zone.conf)

The zone config is really the same format, but different data.
Anything in the [default] zone will be used, unless there is an overriding and matching [view] with a matching AS number or resolver IP address.

The sample data below illustrates a basic name server, 
where the IPv4 address is abstracted to a separate name.

```INI

[default]
send-users.test-ipv6.com:
- FB ipv4.test-ipv6.com


[default]

# Magic record, pulls "send-users" based on ASN
# send-users.test-ipv6.com is the only part of the record that should vary
# based on the source ISP

test-ipv6.com: 
  - SOA ns1.test-ipv6.com. jfesler.test-ipv6.com. 2010050801 10800 3600 604800 86400
  - NS ns1.test-ipv6.com
  - NS ns2.test-ipv6.com
  - MX 10 lists.gigo.com
  - EXPAND send-users.test-ipv6.com
  - SPF "v=spf1 ip4:216.218.228.112/28 ip6:2001:470:1:18::/64 include:_spf.google.com -all"
  - TXT "keybase-site-verification=j96s0SKp4hfmvdR4X0bmFCMzjOTfu56ZhoPXO4VUGa4"

  
ns1.test-ipv6.com: [A 216.218.228.118, AAAA 2001:470:1:18::118]
ns2.test-ipv6.com: [A 209.128.193.197]
ipv4.test-ipv6.com: A 216.218.228.119
ipv6.test-ipv6.com: AAAA 2001:470:1:18::119
mtu1280.test-ipv6.com: AAAA 2001:470:1:18::1280
ds.test-ipv6.com: [EXPAND ipv4.test-ipv6.com, EXPAND ipv6.test-ipv6.com]
```

What you now have is something that will respond to "test-ipv6.com", "ipv4.test-ipv6.com", "ipv6.test-ipv6.com", "ds.test-ipv6.com", and such.  So far so good.

Next, let's `DELEGATE` a subdomain to another name server.   By delegate, we mean - tell any asking name server "I don't have the answer, but I know who does".   This will include the DNS glue records (if we know about them):

```INI
v6ns1.test-ipv6.com: AAAA 2001:470:1:18::119
v6ns.test-ipv6.com: DELEGATE v6ns.test-ipv6.com v6ns1.test-ipv6.com
*.v6ns.test-ipv6.com: EXPAND v6ns.test-ipv6.com
```

Last, you'll perhaps want to start putting in other features.  Like failover, and ISP decisions.   In our case (and, indeed, the whole reason this program exists!), we route traffic for some ISPs to ISP-hosted mirror sites.  These mirrors act identical to the master; they are simply dedicated mirrors on the ISP networks.

```INI
# Comcast operates a pair of mirror nodes for their customers,
# and (on my request) for other ASNs.
[comcast]
as: [11025, 13367, 13385, 20214, 21508, 22258, 22909, 33489, 33490, 33491, 33650, 33651, 33652, 33657, 33659, 33660, 33662, 33667, 33668, 36377, 36733, 53297, 7015, 7016, 7725, 7922]
as: 174  # Cogent
resolver: [50.184.213.245, 2601:9:4e80:199:bd0e:6170:cdf3:49a7]
send-users.test-ipv6.com:
- HC check_mirror comcast-ct.test-ipv6.com
- HC check_mirror comcast-pa.test-ipv6.com
- FB ipv4.test-ipv6.com  

[default]
comcast-ct.test-ipv6.com: A 96.119.0.221
comcast-pa.test-ipv6.com: A 96.119.4.224

```

What that says is:
 * Any requestor from a set of AS numbers, will be in this view.
 * Any resolver in the list of resolers mentioned, will also be in this view.
 * Our previously defined `send-users.test-ipv6.com` will be served differently to anyone behind this view.
 * Health checks are done on the two mirror sites.  If either or both are good, they are substituted in.
 * If both health checks fail, then the original value will be used.

Since "test-ipv6.com" refers to `send-users.test-ipv6.com`, Comcast customers will be magically routed.

 
This means that traffic to "test-ipv6.com" from Comcast goes to the dedicated mirror.

## Zone Resource records

These can come to the right side of a hostname, in the zone configuration.

|RCODE | Example | Purpose|
|------|---------|--------|
|A|A 192.0.2.1|Registers an IPv4 address|
|AAAA|AAAA 2001:db8::1|Registers an IPv6 address|
|CNAME|CNAME example.org|Points to an alternate name.  Will treat as EXPAND if possbile, otherwise just as a normal CNAME.|
|EXPAND|EXPAND server.example.com|Assuming server.example.com is in our local config, expand it and substitute here.  This is like a CNAME except with full expansion before sending to the user.|
|HC|HC check_web server.example.com|Does a health check (using check_web) to make sure that server.example.com is up. If it is up, then it treats it like EXPAND. Otherwise, it is skipped.|
|FB|FB server2.example.com|If we have no other A/AAAA records, then EXPAND serve2.example.com and use as a fallback set of addresses.|

Most other types of well known RRs are parsed.  TTL is not at this time supported.



## Feedback

Jason Fesler <jfesler@gigo.com>
