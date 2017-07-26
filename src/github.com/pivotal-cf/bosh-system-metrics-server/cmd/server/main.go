package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	port := flag.Int("port", 25595, "The port which the grpc metrics server will listen on")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	messages := make(chan *definitions.Event)

	i := ingress.New(bufio.NewReader(os.Stdin), unmarshal.Event, messages)
	e := egress.NewServer(messages)

	grpcServer := grpc.NewServer()
	definitions.RegisterEgressServer(grpcServer, e)

	stopReadingMessages := i.Start()
	stopWritingMessages := e.Start()

	defer func() {
		stopReadingMessages()
		stopWritingMessages()
	}()

	log.Printf("bosh system metrics grpc server listening on %d\n", *port)
	grpcServer.Serve(lis)
}
