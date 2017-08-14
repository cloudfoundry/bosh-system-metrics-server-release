package monitor

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"expvar"
)

// New creates a health metrics server
func NewHealth(port uint32) Starter {
	return &profiler{port}
}

type health struct {
	port uint32
}

// Start initializes a monitor health server
func (s *health) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		log.Printf("unable to start monitor endpoint: %s", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", expvar.Handler())

	fmt.Printf("starting monitor endpoint on http://%s/health\n", lis.Addr().String())
	err = http.Serve(lis, mux)
	log.Printf("error starting the monitor server: %s", err)
}
