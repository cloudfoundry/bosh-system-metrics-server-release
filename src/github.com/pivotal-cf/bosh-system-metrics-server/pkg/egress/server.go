package egress

import (
	"log"

	"expvar"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
)

type BoshMetricsServer struct {
	messages chan *definitions.Event
}

var (
	egressSent *expvar.Int
)

func init() {
	egressSent = expvar.NewInt("egress.sent")
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
	for event := range s.messages {
		err := srv.Send(event)
		if err != nil {
			log.Printf("Send Error: %s", err)
			return err
		}
		egressSent.Add(1)
	}

	return nil
}
