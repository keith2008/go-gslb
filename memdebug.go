package main

import (
	"fmt"
	"log"
	"runtime"
)

type memDebugType struct {
	m1     runtime.MemStats
	m2     runtime.MemStats
	prefix string
}

func startMemDebugString(s string) (md *memDebugType) {
	md = new(memDebugType)
	md.prefix = s
	runtime.ReadMemStats(&md.m1) // Do at the last possible moment; don't count OUR memory needs
	return md
}

func startMemDebug() (md *memDebugType) {
	pc, file, line, _ := runtime.Caller(1)
	f1 := runtime.FuncForPC(pc)
	funcname := f1.Name()
	prefix := fmt.Sprintf("ReadMemStats %v(%v:%v)", funcname, file, line)
	return (startMemDebugString(prefix))
}
func (md *memDebugType) finishMemDebug() {
	runtime.ReadMemStats(&md.m2)

	if md.m1.Alloc != md.m2.Alloc {
		log.Printf("%s %s %v\n", md.prefix, "Alloc", md.m2.Alloc-md.m1.Alloc)
	}
	if md.m1.TotalAlloc != md.m2.TotalAlloc {
		log.Printf("%s %s %v\n", md.prefix, "TotalAlloc", md.m2.TotalAlloc-md.m1.TotalAlloc)
	}
	if md.m1.Sys != md.m2.Sys {
		log.Printf("%s %s %v\n", md.prefix, "Sys", md.m2.Sys-md.m1.Sys)
	}
	if md.m1.Lookups != md.m2.Lookups {
		log.Printf("%s %s %v\n", md.prefix, "Lookups", md.m2.Lookups-md.m1.Lookups)
	}
	if md.m1.Mallocs != md.m2.Mallocs {
		log.Printf("%s %s %v\n", md.prefix, "Mallocs", md.m2.Mallocs-md.m1.Mallocs)
	}
	if md.m1.Frees != md.m2.Frees {
		log.Printf("%s %s %v\n", md.prefix, "Frees", md.m2.Frees-md.m1.Frees)
	}
	if md.m1.HeapAlloc != md.m2.HeapAlloc {
		log.Printf("%s %s %v\n", md.prefix, "HeapAlloc", md.m2.HeapAlloc-md.m1.HeapAlloc)
	}
	if md.m1.HeapSys != md.m2.HeapSys {
		log.Printf("%s %s %v\n", md.prefix, "HeapSys", md.m2.HeapSys-md.m1.HeapSys)
	}
	if md.m1.HeapIdle != md.m2.HeapIdle {
		log.Printf("%s %s %v\n", md.prefix, "HeapIdle", md.m2.HeapIdle-md.m1.HeapIdle)
	}
	if md.m1.HeapInuse != md.m2.HeapInuse {
		log.Printf("%s %s %v\n", md.prefix, "HeapInuse", md.m2.HeapInuse-md.m1.HeapInuse)
	}
	if md.m1.HeapReleased != md.m2.HeapReleased {
		log.Printf("%s %s %v\n", md.prefix, "HeapReleased", md.m2.HeapReleased-md.m1.HeapReleased)
	}
	if md.m1.HeapObjects != md.m2.HeapObjects {
		log.Printf("%s %s %v\n", md.prefix, "HeapObjects", md.m2.HeapObjects-md.m1.HeapObjects)
	}
	if md.m1.StackInuse != md.m2.StackInuse {
		log.Printf("%s %s %v\n", md.prefix, "StackInuse", md.m2.StackInuse-md.m1.StackInuse)
	}
	if md.m1.StackSys != md.m2.StackSys {
		log.Printf("%s %s %v\n", md.prefix, "StackSys", md.m2.StackSys-md.m1.StackSys)
	}
	if md.m1.MSpanInuse != md.m2.MSpanInuse {
		log.Printf("%s %s %v\n", md.prefix, "MSpanInuse", md.m2.MSpanInuse-md.m1.MSpanInuse)
	}
	if md.m1.MSpanSys != md.m2.MSpanSys {
		log.Printf("%s %s %v\n", md.prefix, "MSpanSys", md.m2.MSpanSys-md.m1.MSpanSys)
	}
	if md.m1.MCacheInuse != md.m2.MCacheInuse {
		log.Printf("%s %s %v\n", md.prefix, "MCacheInuse", md.m2.MCacheInuse-md.m1.MCacheInuse)
	}
	if md.m1.MCacheSys != md.m2.MCacheSys {
		log.Printf("%s %s %v\n", md.prefix, "MCacheSys", md.m2.MCacheSys-md.m1.MCacheSys)
	}
	if md.m1.BuckHashSys != md.m2.BuckHashSys {
		log.Printf("%s %s %v\n", md.prefix, "BuckHashSys", md.m2.BuckHashSys-md.m1.BuckHashSys)
	}
	if md.m1.GCSys != md.m2.GCSys {
		log.Printf("%s %s %v\n", md.prefix, "GCSys", md.m2.GCSys-md.m1.GCSys)
	}
	if md.m1.OtherSys != md.m2.OtherSys {
		log.Printf("%s %s %v\n", md.prefix, "OtherSys", md.m2.OtherSys-md.m1.OtherSys)
	}
	if md.m1.NextGC != md.m2.NextGC {
		log.Printf("%s %s %v\n", md.prefix, "NextGC", md.m2.NextGC-md.m1.NextGC)
	}
	if md.m1.LastGC != md.m2.LastGC {
		log.Printf("%s %s %v\n", md.prefix, "LastGC", md.m2.LastGC-md.m1.LastGC)
	}
	if md.m1.PauseTotalNs != md.m2.PauseTotalNs {
		log.Printf("%s %s %v\n", md.prefix, "PauseTotalNs", md.m2.PauseTotalNs-md.m1.PauseTotalNs)
	}
	if md.m1.NumGC != md.m2.NumGC {
		log.Printf("%s %s %v\n", md.prefix, "NumGC", md.m2.NumGC-md.m1.NumGC)
	}

}
