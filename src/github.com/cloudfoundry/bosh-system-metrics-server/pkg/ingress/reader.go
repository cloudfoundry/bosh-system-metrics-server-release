package ingress

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"expvar"

	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/definitions"
)

type unmarshaller func(eventJSON []byte) (*definitions.Event, error)

type Ingestor struct {
	port         int
	unmarshaller unmarshaller
	output       chan *definitions.Event
}

var (
	ingressReceivedCounter      *expvar.Int
	ingressUnmarshallErrCounter *expvar.Int
	ingressReadErrCounter       *expvar.Int
)

func init() {
	ingressReceivedCounter = expvar.NewInt("ingress.received")
	ingressUnmarshallErrCounter = expvar.NewInt("ingress.unmarshall_err")
	ingressReadErrCounter = expvar.NewInt("ingress.read_err")
}

// New returns a new Ingestor.
func New(p int, u unmarshaller, m chan *definitions.Event) *Ingestor {
	return &Ingestor{
		port:         p,
		unmarshaller: u,
		output:       m,
	}
}

// Start spins up a go routine to listen for bosh events over tcp.
// It returns a shutdown function that shuts down any open connections
// and closes the listener.
func (i *Ingestor) Start() func() {
	stop := make(chan struct{})
	ingressLis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", i.port))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", i.port, err)
	}
	log.Printf("ingestor listening on %s\n", ingressLis.Addr().String())

	go func() {
		for {
			conn, err := ingressLis.Accept()
			if err != nil {
				return
			}

			go i.handleConnection(conn, stop)
		}
	}()

	return func() {
		ingressLis.Close()
		close(stop)
	}
}

func (i *Ingestor) handleConnection(conn net.Conn, stop chan struct{}) {
	reader := bufio.NewReader(conn)
	for {
		b, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("error reading: %s\n", err)
			ingressReadErrCounter.Add(1)
			return
		}

		evt, err := i.unmarshaller(b)
		if err != nil {
			log.Printf("error unmarshalling: %s\n", err)
			ingressUnmarshallErrCounter.Add(1)
			continue
		}

		if shouldStop(stop) {
			return
		} else {
			i.output <- evt
			ingressReceivedCounter.Add(1)
		}
	}
}

func shouldStop(s chan struct{}) bool {
	select {
	case <-s:
		return true
	default:
		return false
	}
}
