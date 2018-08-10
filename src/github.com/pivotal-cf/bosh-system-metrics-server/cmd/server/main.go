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

	"crypto/x509"
	"io/ioutil"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/monitor"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/tokenchecker"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	egressPort := flag.Int("egress-port", 25595, "The port which the grpc metrics server will listen on")
	ingressPort := flag.Int("ingress-port", 25594, "The port listening for bosh system events")
	certPath := flag.String("metrics-cert", "", "The public cert for the metrics server")
	keyPath := flag.String("metrics-key", "", "The private key for the metrics server")

	uaaURL := flag.String("uaa-url", "", "The UAA URL")
	uaaCA := flag.String("uaa-ca", "", "The path to the UAA CA cert")
	uaaClient := flag.String("uaa-client-identity", "", "The UAA client identity which has access to check token")
	uaaPassword := flag.String("uaa-client-password", "", "The UAA client secret which has access to check token")

	healthPort := flag.Int("health-port", 0, "The port for the localhost health endpoint")
	pprofPort := flag.Int("pprof-port", 0, "The port for the localhost pprof endpoint")

	flag.Parse()

	tlsConfig, err := newTLSConfig(*certPath, *keyPath)
	if err != nil {
		log.Fatalf("unable to parse certs: %s", err)
	}

	egressLis, err := net.Listen("tcp", fmt.Sprintf(":%d", *egressPort))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", *egressPort, err)
	}

	uaaTLSConfig := &tls.Config{}
	err = setCACert(uaaTLSConfig, *uaaCA)
	if err != nil {
		log.Fatal(err)
	}
	tokenChecker := tokenchecker.New(&tokenchecker.TokenCheckerConfig{
		UaaURL:      *uaaURL,
		TLSConfig:   uaaTLSConfig,
		UaaClient:   *uaaClient,
		UaaPassword: *uaaPassword,
		Authority:   "bosh.system_metrics.read",
	})

	messages := make(chan *definitions.Event, 10000)

	i := ingress.New(*ingressPort, unmarshal.Event, messages)
	e := egress.NewServer(messages, tokenChecker)

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time: 1 * time.Minute,
		}),
	)
	definitions.RegisterEgressServer(grpcServer, e)

	stopReadingMessages := i.Start()
	stopWritingMessages := e.Start()

	defer func() {
		fmt.Println("process shutting down, stop accepting messages from bosh health monitor...")
		stopReadingMessages()
		close(messages)

		fmt.Println("drain remaining messages...")
		stopWritingMessages()
		grpcServer.GracefulStop()

		fmt.Println("DONE")
	}()

	go monitor.NewHealth(uint32(*healthPort)).Start()
	go monitor.NewProfiler(uint32(*pprofPort)).Start()

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

func setCACert(tlsConfig *tls.Config, caPath string) error {
	caCertBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return fmt.Errorf("cannot parse ca cert from %s", caPath)
	}

	tlsConfig.RootCAs = caCertPool

	return nil
}
