package egress

import (
	"log"

	"expvar"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"google.golang.org/grpc/metadata"
)

type BoshMetricsServer struct {
	messages chan *definitions.Event
}

var (
	egressSentCounter    *expvar.Int
	egressSendErrCounter *expvar.Int
)

func init() {
	egressSentCounter = expvar.NewInt("egress.sent")
	egressSendErrCounter = expvar.NewInt("egress.send_err")
}

func NewServer(m chan *definitions.Event) *BoshMetricsServer {
	return &BoshMetricsServer{
		messages: m,
	}
}

func (s *BoshMetricsServer) Start() func() {
	return func() {}
}

func (s *BoshMetricsServer) BoshMetrics(r *definitions.EgressRequest, srv definitions.Egress_BoshMetricsServer) error {
	md, _ := metadata.FromContext(srv.Context())
	token, ok := md["authorization"]
	if ok {
		log.Printf("Got token from client: %+v", token)
	}

	for event := range s.messages {
		err := srv.Send(event)
		if err != nil {
			log.Printf("Send Error: %s", err)
			egressSendErrCounter.Add(1)
			return err
		}
		egressSentCounter.Add(1)
	}

	return nil
}
