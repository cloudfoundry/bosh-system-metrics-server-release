package egress

import (
	"log"

	"expvar"

	"errors"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
)

var (
	authorizationMissingErr = errors.New("Request does not include authorization token")
	invalidAuthErr          = func(err error) error {
		return status.Errorf(codes.PermissionDenied, "Authorization token is invalid. It must include the bosh.system_metrics.read authority and not be expired: %s", err)
	}
)

type BoshMetricsServer struct {
	messages     chan *definitions.Event
	tokenChecker tokenChecker
}

var (
	egressSentCounter    *expvar.Int
	egressSendErrCounter *expvar.Int
	egressAuthErrCounter *expvar.Int
)

func init() {
	egressSentCounter = expvar.NewInt("egress.sent")
	egressSendErrCounter = expvar.NewInt("egress.send_err")
	egressAuthErrCounter = expvar.NewInt("egress.auth_err")
}

type tokenChecker interface {
	CheckToken(token string) error
}

func NewServer(m chan *definitions.Event, t tokenChecker) *BoshMetricsServer {
	return &BoshMetricsServer{
		messages:     m,
		tokenChecker: t,
	}
}

func (s *BoshMetricsServer) Start() func() {
	return func() {}
}

func (s *BoshMetricsServer) BoshMetrics(r *definitions.EgressRequest, srv definitions.Egress_BoshMetricsServer) error {
	md, ok := metadata.FromIncomingContext(srv.Context())
	if !ok {
		egressAuthErrCounter.Add(1)
		return authorizationMissingErr
	}

	tokens := md["authorization"]
	if len(tokens) == 0 {
		egressAuthErrCounter.Add(1)
		return authorizationMissingErr
	}

	err := s.tokenChecker.CheckToken(tokens[0])
	if err != nil {
		egressAuthErrCounter.Add(1)
		return invalidAuthErr(err)
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
