package main

import (
	"expvar"
)

var statsQuery = expvar.NewMap("query")
var statsResponse = expvar.NewMap("response")
var statsQname = expvar.NewMap("qname")
var statsMaxMind = expvar.NewMap("maxmind")

func init() {

}
