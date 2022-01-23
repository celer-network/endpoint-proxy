package main

import (
	"flag"

	"github.com/celer-network/endpoint-proxy/endpointproxy"
	"github.com/celer-network/goutils/log"
)

var (
	port     = flag.Int("p", 10090, "port for proxy")
	chainId  = flag.Uint64("cid", 1666700000, "chain id")
	endpoint = flag.String("endpoint", "https://api.s0.b.hmny.io", "origin endpoint url")
)

func main() {
	flag.Parse()
	if *port <= 0 {
		log.Fatalln("invalid port")
	}
	if *chainId <= 0 {
		log.Fatalln("invalid chainId")
	}
	if *endpoint == "" {
		log.Fatalln("invalid endpoint")
	}
	// initialize a reverse proxy and pass the actual backend server url here
	err := endpointproxy.StartProxy(*endpoint, *chainId, *port)
	if err != nil {
		panic(err)
	}
	select {}
}
