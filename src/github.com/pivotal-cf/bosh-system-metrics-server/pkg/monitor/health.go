package monitor

import (
	"expvar"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
)

// New creates a health metrics server
func NewHealth(port uint32) Starter {
	return &health{port}
}

type health struct {
	port uint32
}

// Start initializes a monitor health server
func (s *health) Start() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		log.Printf("unable to start health endpoint: %s", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", expvar.Handler())

	fmt.Printf("starting health endpoint on http://%s/health\n", lis.Addr().String())
	err = http.Serve(lis, mux)
	log.Printf("error starting the health server: %s", err)
}
