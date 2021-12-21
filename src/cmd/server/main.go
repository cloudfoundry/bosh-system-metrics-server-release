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

	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/config"
	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/definitions"
	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/egress"
	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/ingress"
	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/monitor"
	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/tokenchecker"
	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/unmarshal"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

var defaultServerCipherSuites = []uint16{
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

func main() {
	rand.Seed(time.Now().UnixNano())

	configFilePath := flag.String("config", "", "A path to the configuration file")

	flag.Parse()

	c, err := config.Read(*configFilePath)
	if err != nil {
		log.Fatalf("unable to parse config: %s", err)
	}

	tlsConfig, err := newTLSConfig(c.CertPath, c.KeyPath)
	if err != nil {
		log.Fatalf("unable to parse certs: %s", err)
	}

	egressLis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.EgressPort))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", c.EgressPort, err)
	}

	uaaTLSConfig := &tls.Config{}
	err = setCACert(uaaTLSConfig, c.UaaCA)
	if err != nil {
		log.Fatal(err)
	}

	tokenChecker := tokenchecker.New(&tokenchecker.TokenCheckerConfig{
		UaaURL:      c.UaaURL,
		TLSConfig:   uaaTLSConfig,
		UaaClient:   c.UaaClientIdentity,
		UaaPassword: c.UaaClientPassword,
		Authority:   "bosh.system_metrics.read",
	})

	messages := make(chan *definitions.Event, 10000)

	i := ingress.New(c.IngressPort, unmarshal.Event, messages)
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

	go monitor.NewHealth(uint32(c.HealthPort)).Start()
	go monitor.NewProfiler(uint32(c.PProfPort)).Start()

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
		CipherSuites:       defaultServerCipherSuites,
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

func getUaaPassword(filePath string) (string, error) {
	passwordBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(passwordBytes), nil
}
