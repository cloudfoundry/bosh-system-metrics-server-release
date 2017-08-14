package monitor

import (
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
)

// Starter is something that can be started
type Starter interface {
	Start()
}

// New creates a monitor profiler
func NewProfiler(port uint32) Starter {
	return &profiler{port}
}

type profiler struct {
	port uint32
}

// Start initializes a monitor profiler on a port
func (s *profiler) Start() {
	addr := fmt.Sprintf("localhost:%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("error creating pprof listener: %s", err)
	}

	log.Printf("starting pprof profiler on: %s", lis.Addr().String())
	err = http.Serve(lis, nil)
	if err != nil {
		log.Panicf("error starting pprof profiler: %s", err)
	}
}
