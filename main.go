package main

import (
	// "gigo.com/gslb/conf"
	// "gigo.com/gslb/maxmind"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Profiling

var etcFlag = flag.String("etc", "etc", "Config directory")
var debugFlag = flag.Bool("debug", false, "show extra output to screen")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write cpu profile to file")
var profile = flag.Bool("profile", false, "export profiler to port 28000")
var httpOption = flag.String("http", "", "Start HTTP server, ie: :28000")

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func webServerTips(where string) {
	h := ourHostname()
	if strings.HasPrefix(where, "[::]:") {
		where = h + where[4:]
	}

	log.Printf(`Tips:
		
30 seconds of CPU
go tool pprof http://%s/debug/pprof/profile

Memory Heap
go tool pprof http://%s/debug/pprof/heap
go tool pprof --inuse_objects http://%s/debug/pprof/heap

goroutines 
go tool pprof http://%s/debug/pprof/goroutine?

Stats
curl --silent http://%s/debug/vars | grep -vi malloc

`, where, where, where, where, where)

}

func startOneHTTP(addr string) (bound string, err error) {
	log.Printf("startOneHTTP(%s)\n", addr)
	sock, err := net.Listen("tcp", addr)
	if err != nil {
		return addr, err
	}
	go func() {
		http.Serve(sock, nil)
	}()
	return sock.Addr().String(), nil
}
func startHTTP() {
	log.Printf("startHTTP\n")
	tries := []string{*httpOption}
	if found, ok := GlobalConfig().GetSectionNameValueStrings("server", "http"); ok {
		tries = append(tries, found...)
	}
	first := true
	for _, try := range tries {
		if try != "" {
			bound, err := startOneHTTP(try)
			if err == nil {
				log.Printf("HTTP listening on %v\n", bound)
				if first {
					first = false
					webServerTips(bound)
				}
			} else {
				log.Printf("HTTP failed on %v: %v", try, err)

			}
		}
	}
}

func main() {
	flag.Parse()
	log.Printf("EtcFlag is %v\n", *etcFlag)
	log.Printf("DebugFlag is %v\n", *debugFlag)
	log.Printf("main()\n")
	initGlobal(*etcFlag)
	startHTTP()
	startDNS()
	log.Printf("Sitting and waiting ()\n")

	// Who wants to live forever?
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
forever:
	for {
		select {
		case s := <-sig:
			log.Printf("Signal (%#v) received, stopping\n", s)
			break forever
		}
	}

}
