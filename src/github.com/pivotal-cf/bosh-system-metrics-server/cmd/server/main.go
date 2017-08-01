package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"

	"crypto/tls"

	"expvar"
	"net/http"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
	"google.golang.org/grpc/credentials"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	egressPort := flag.Int("egress-port", 25595, "The port which the grpc metrics server will listen on")
	ingressPort := flag.Int("ingress-port", 25594, "The port listening for bosh system events")
	certPath := flag.String("metrics-cert", "", "The public cert for the metrics server")
	keyPath := flag.String("metrics-key", "", "The private key for the metrics server")

	healthPort := flag.Int("health-port", 19110, "The port for the localhost health endpoint")
	flag.Parse()

	tlsConfig, err := newTLSConfig(*certPath, *keyPath)
	if err != nil {
		log.Fatalf("unable to parse certs: %s", err)
	}

	egressLis, err := net.Listen("tcp", fmt.Sprintf(":%d", *egressPort))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", *egressPort, err)
	}

	messages := make(chan *definitions.Event)

	i := ingress.New(*ingressPort, unmarshal.Event, messages)
	e := egress.NewServer(messages)

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	definitions.RegisterEgressServer(grpcServer, e)

	stopReadingMessages := i.Start()
	stopWritingMessages := e.Start()

	defer func() {
		stopReadingMessages()
		stopWritingMessages()
		egressLis.Close()
	}()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/health", expvar.Handler())
		http.ListenAndServe(fmt.Sprintf("localhost:%d", *healthPort), mux)
	}()

	log.Printf("bosh system metrics grpc server listening on %s\n", egressLis.Addr().String())
	err = grpcServer.Serve(egressLis)
	if err != nil {
		log.Fatalf("unable to serve grpc server: %s", err)
	}
}

func newTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load keypair: %s", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	tlsConfig.Certificates = []tls.Certificate{tlsCert}

	return tlsConfig, err
}
