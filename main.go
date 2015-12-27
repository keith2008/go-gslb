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
	"runtime/pprof"
	"strings"
	"syscall"
)

// Profiling

var etcFlag = flag.String("etc", "etc", "Config directory")
var debugFlag = flag.Bool("debug", false, "show extra output to screen")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write cpu profile to file")
var profile = flag.Bool("profile", false, "export profiler to port 28000")

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
	log.Printf(`Tips:
30 seconds of CPU
go tool pprof http://%s/debug/pprof/profile

Memory Heap
go tool pprof http://%s/debug/pprof/heap
go tool pprof --inuse_objects http://%s/debug/pprof/heap

goroutines 
go tool pprof http://%s/debug/pprof/goroutine?
`, where, where, where, where)

}
func webServerStart(where string) {
	go func() {
		log.Println(http.ListenAndServe(where, nil))
	}()
}
func webServer() {
	h := ourHostname()

	if strings.HasSuffix(h, ".local") {
		where := "127.0.0.1:28000"
		webServerStart(where)
		webServerTips(where)
	} else {
		where := getLocalIP() + ":28000"
		webServerStart(where)
		webServerTips(where)
	}
}

func main() {
	flag.Parse()
	log.Printf("EtcFlag is %v\n", *etcFlag)
	log.Printf("DebugFlag is %v\n", *debugFlag)
	log.Printf("cpuprofile is %v\n", *cpuprofile)
	log.Printf("memprofile is %v\n", *memprofile)
	log.Printf("(web)profile is %v\n", *profile)

	// Profiling MUST Be in main or else the defer will close prematurely
	// Jasons-MacBook:gslb jfesler$ ./gslb -cpuprofile cpu.pprof
	// Jasons-MacBook:gslb jfesler$ go tool pprof --pdf gslb cpu.pprof > 1.pdf
	// http://blog.golang.org/profiling-go-programs
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Println("Error: ", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Println("Error: ", err)
		}
		pprof.WriteHeapProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *profile {
		webServer()
	}

	log.Printf("main()\n")
	initGlobal(*etcFlag)
	startDNS()

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
