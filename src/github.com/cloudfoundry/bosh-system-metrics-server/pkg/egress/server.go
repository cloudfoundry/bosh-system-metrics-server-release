package egress

import (
	"log"

	"expvar"

	"errors"

	"sync"

	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/definitions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

	wg sync.WaitGroup

	mu                     sync.RWMutex
	registry               map[string]chan *definitions.Event
	subscriptionBufferSize int
}

var (
	egressSubscriptionSent    *expvar.Map
	egressSendErrCounter      *expvar.Int
	egressAuthErrCounter      *expvar.Int
	egressSubscriptionDropped *expvar.Map
	egressProcessedCounter    *expvar.Int
)

func init() {
	egressSubscriptionSent = expvar.NewMap("egress.subscription_sent")
	egressSendErrCounter = expvar.NewInt("egress.send_err")
	egressAuthErrCounter = expvar.NewInt("egress.auth_err")
	egressSubscriptionDropped = expvar.NewMap("egress.subscription_dropped")
	egressProcessedCounter = expvar.NewInt("egress.processed")
}

type tokenChecker interface {
	CheckToken(token string) error
}

type ServerOpt func(*BoshMetricsServer)

func WithSubscriptionBufferSize(n int) ServerOpt {
	return func(s *BoshMetricsServer) {
		s.subscriptionBufferSize = n
	}
}

// NewServer returns a BoshMetricsServer.
// It serves bosh metrics via a grpc connections from clients.
func NewServer(m chan *definitions.Event, t tokenChecker, opts ...ServerOpt) *BoshMetricsServer {
	s := &BoshMetricsServer{
		messages:               m,
		registry:               make(map[string]chan *definitions.Event),
		tokenChecker:           t,
		subscriptionBufferSize: 1024,
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

// Start spins up a new go routine that distributes metrics to each subscription.
// It returns a shutdown function which blocks until all subscriptions are
// drained.
func (s *BoshMetricsServer) Start() func() {
	done := make(chan struct{})

	go func() {
		for message := range s.messages {
			s.mu.RLock()
			for subscription, ch := range s.registry {
				select {
				case ch <- message:
				default:
					egressSubscriptionDropped.Add(subscription, 1)
				}
			}
			egressProcessedCounter.Add(1)
			s.mu.RUnlock()
		}
		close(done)
	}()

	return func() {
		<-done

		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, subscriptionMsgs := range s.registry {
			close(subscriptionMsgs)
		}

		s.wg.Wait()
	}
}

// BoshMetrics is the grpc handler that serves EgressRequests.
// It verifies auth tokens from the `authorization` metadata.
// It returns an error if the auth token is missing or invalid.
func (s *BoshMetricsServer) BoshMetrics(r *definitions.EgressRequest, srv definitions.Egress_BoshMetricsServer) error {
	err := s.checkToken(srv)
	if err != nil {
		return err
	}

	s.wg.Add(1)
	defer s.wg.Done()

	m := s.register(r.SubscriptionId)
	for event := range m {
		err := srv.Send(event)
		if err != nil {
			log.Printf("Send Error: %s\n", err)
			egressSendErrCounter.Add(1)
			retryMessageOnSubscription(m, event, r.SubscriptionId)
			return err
		}
		egressSubscriptionSent.Add(r.SubscriptionId, 1)
	}

	return nil
}

// TODO: Change retry strategy. The subscription channel
// should be read only at this point. We could get a send on close channel
// if this happens after shutdown.
func retryMessageOnSubscription(subscriptionMsgs chan *definitions.Event, event *definitions.Event, subscription string) {
	select {
	case subscriptionMsgs <- event:
	default:
		egressSubscriptionDropped.Add(subscription, 1)
	}
}

func (s *BoshMetricsServer) register(subscriptionId string) chan *definitions.Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs, ok := s.registry[subscriptionId]
	if !ok {
		msgs = make(chan *definitions.Event, s.subscriptionBufferSize)
		s.registry[subscriptionId] = msgs
	}

	return msgs
}

func (s *BoshMetricsServer) checkToken(srv definitions.Egress_BoshMetricsServer) error {
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

	return nil
}
