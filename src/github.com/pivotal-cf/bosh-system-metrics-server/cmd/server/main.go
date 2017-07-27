package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	egressPort := flag.Int("egress-port", 25595, "The port which the grpc metrics server will listen on")
	ingressPort := flag.Int("ingress-port", 25594, "The port listening for bosh system events")
	flag.Parse()

	egressLis, err := net.Listen("tcp", fmt.Sprintf(":%d", *egressPort))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", *egressPort, err)
	}

	messages := make(chan *definitions.Event)

	i := ingress.New(*ingressPort, unmarshal.Event, messages)
	e := egress.NewServer(messages)

	grpcServer := grpc.NewServer()
	definitions.RegisterEgressServer(grpcServer, e)

	stopReadingMessages := i.Start()
	stopWritingMessages := e.Start()

	defer func() {
		stopReadingMessages()
		stopWritingMessages()
		egressLis.Close()
	}()

	log.Printf("bosh system metrics grpc server listening on %s\n", egressLis.Addr().String())
	err = grpcServer.Serve(egressLis)
	if err != nil {
		log.Fatalf("unable to serve grpc server: %s", err)
	}
}
