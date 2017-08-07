package egress_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc"

	"time"

	"io/ioutil"
	"log"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"sync"
)

func TestBoshMetricsWritesEventSuccessfully(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	sender := newSpyEgressSender(validContext("test-token"), 50)
	tokenChecker := newSpyTokenChecker(nil)
	server := egress.NewServer(messages, tokenChecker)
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	go server.BoshMetrics(req, sender)
	time.Sleep(time.Millisecond * 100)

	messages <- event

	server.Start()

	Eventually(sender.received).Should(Receive(Equal(event)))
	Eventually(tokenChecker.received).Should(Receive(Equal("test-token")))
}

func TestBoshMetricsReturnsErrorWhenUnableToSend(t *testing.T) {
	RegisterTestingT(t)
	log.SetOutput(ioutil.Discard)

	messages := make(chan *definitions.Event, 1000)
	sender := newSpyEgressSender(validContext("test-token"), 50)
	sender.SendError(errors.New("unable to send"))
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	var returnErr error
	done := make(chan struct{})
	go func() {
		returnErr = server.BoshMetrics(req, sender)
		close(done)
	}()
	time.Sleep(time.Millisecond * 100)

	messages <- event

	server.Start()

	<-done

	Expect(sender.received).To(BeEmpty())
	Expect(returnErr).ToNot(BeNil())
}

func TestBoshMetricsWithoutIncomingContext(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	invalidContext := context.Background()
	sender := newSpyEgressSender(invalidContext, 50)
	sender.SendError(errors.New("unable to send"))
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	err := server.BoshMetrics(req, sender)

	Expect(err).To(HaveOccurred())
}

func TestBoshMetricsWhenTokenIsInvalid(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	sender := newSpyEgressSender(validContext("test-token"), 50)
	tokenChecker := newSpyTokenChecker(errors.New("token-invalid"))
	server := egress.NewServer(messages, tokenChecker)
	req := &definitions.EgressRequest{}

	err := server.BoshMetrics(req, sender)

	Expect(err).To(HaveOccurred())

	st, _ := status.FromError(err)
	Expect(st.Code()).To(Equal(codes.PermissionDenied))
}

func TestBoshMetricsWithoutAuthorizationHeader(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	contextWithNoHeader := metadata.NewIncomingContext(context.Background(), metadata.New(nil))
	sender := newSpyEgressSender(contextWithNoHeader, 50)
	sender.SendError(errors.New("unable to send"))
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{}

	err := server.BoshMetrics(req, sender)

	Expect(err).To(HaveOccurred())
}

func TestBoshMetricsDividesEventsBetweenMultipleClientsWithSameSubscriptionIds(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	sender1 := newSpyEgressSender(validContext("test-token-a"), 50)
	sender2 := newSpyEgressSender(validContext("test-token-b"), 50)
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	go server.BoshMetrics(req, sender1)
	go server.BoshMetrics(req, sender2)
	time.Sleep(time.Millisecond * 100)

	server.Start()

	for i := 0; i < 100; i++ {
		messages <- event
	}

	Eventually(func() int { return len(sender1.received) + len(sender2.received) }).Should(Equal(100))
	Expect(len(sender1.received)).To(BeNumerically(">", 0))
	Expect(len(sender2.received)).To(BeNumerically(">", 0))
}

func TestBoshMetricsDuplicatesEventsBetweenMultipleClientsWithDifferentSubscriptionIds(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	sender1 := newSpyEgressSender(validContext("test-token-a"), 50)
	sender2 := newSpyEgressSender(validContext("test-token-b"), 50)
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req1 := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}
	req2 := &definitions.EgressRequest{SubscriptionId: "subscriptionB"}

	go server.BoshMetrics(req1, sender1)
	go server.BoshMetrics(req2, sender2)
	time.Sleep(time.Millisecond * 100)

	server.Start()

	for i := 0; i < 50; i++ {
		messages <- event
	}

	Eventually(func() int { return len(sender1.received) }, "2s").Should(Equal(50))
	Eventually(func() int { return len(sender2.received) }, "2s").Should(Equal(50))
}

func TestBoshMetricsRedirectsAllEventsToRemainingConnectedClients(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	sender1 := newSpyEgressSender(validContext("test-token-a"), 50)
	sender2 := newSpyEgressSender(validContext("test-token-a"), 50)
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req1 := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}
	req2 := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	go server.BoshMetrics(req1, sender1)
	go server.BoshMetrics(req2, sender2)
	// giving the senders a chance to register their subscriptions
	time.Sleep(time.Millisecond * 100)

	server.Start()
	for i := 0; i < 50; i++ {
		messages <- event

		if i == 15 {
			// Kill one of the senders midway.
			sender1.SendError(errors.New("unable to send"))
		}
	}

	Eventually(func() int { return len(sender1.received) + len(sender2.received) }, "2s").Should(Equal(50))
	Expect(len(sender2.received)).To(BeNumerically(">", len(sender1.received)))
}

func TestDrainAndDie(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 1000)
	sender1 := newSpyEgressSender(validContext("test-token-a"), 50)
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req1 := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	go server.BoshMetrics(req1, sender1)
	time.Sleep(time.Millisecond * 100)

	for i := 0; i < 1000; i++ {
		messages <- event
	}

	stop := server.Start()
	close(messages)
	stop()

	Expect(len(messages)).To(Equal(0))
}

func TestSlowClientsDoesNotAffectFastClients(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 20)
	sender1 := newSpyEgressSender(validContext("test-token-a"), 100, withSendRate(time.Millisecond))
	sender2 := newSpyEgressSender(validContext("test-token-b"), 100, withSendRate(time.Second))
	// SubscriptionBufferSize is less than number of messages to get message distribution in Start() blocked.
	server := egress.NewServer(messages, newSpyTokenChecker(nil), egress.WithSubscriptionBufferSize(10))
	req1 := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}
	req2 := &definitions.EgressRequest{SubscriptionId: "subscriptionB"}

	go server.BoshMetrics(req1, sender1)
	go server.BoshMetrics(req2, sender2)
	// This sleep is so we can ensure that the senders
	// are registered before we start sending messages
	time.Sleep(time.Millisecond * 100)

	server.Start()

	go func() {
		// We send messages at a slower rate than the fastest sender to avoid
		// dropping its messages so we can get an accurate sender.received count
		for i := 0; i < 20; i++ {
			messages <- event
			time.Sleep(3 * time.Millisecond)
		}
	}()

	Eventually(func() int { return len(sender1.received) }, "2s").Should(Equal(20))
	Expect(len(sender2.received)).To(BeNumerically("<", 20))
}

// ------ SPIES ------
type spyEgressSender struct {
	received       chan *definitions.Event
	token          string
	context        context.Context
	ingestRate     time.Duration
	sendBufferSize int

	mu        sync.Mutex
	sendError error

	grpc.ServerStream
}

type EgressSenderOpt func(*spyEgressSender)

func withSendRate(d time.Duration) EgressSenderOpt {
	return func(s *spyEgressSender) {
		s.ingestRate = d
	}
}

func newSpyEgressSender(ctx context.Context, bufferSize int, opts ...EgressSenderOpt) *spyEgressSender {
	s := &spyEgressSender{
		received:   make(chan *definitions.Event, bufferSize),
		context:    ctx,
		ingestRate: time.Millisecond,
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *spyEgressSender) Context() context.Context {
	return s.context
}

func (s *spyEgressSender) Send(e *definitions.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sendError != nil {
		return s.sendError
	}
	time.Sleep(s.ingestRate)

	s.received <- e
	return nil
}

func (s *spyEgressSender) SendError(e error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sendError = e
}

type spyTokenChecker struct {
	err      error
	received chan string
}

func newSpyTokenChecker(e error) *spyTokenChecker {
	return &spyTokenChecker{
		err:      e,
		received: make(chan string, 100),
	}
}

func (t *spyTokenChecker) CheckToken(token string) error {
	t.received <- token
	return t.err
}

func validContext(token string) context.Context {
	md := metadata.New(map[string]string{
		"authorization": token,
	})

	return metadata.NewIncomingContext(context.Background(), md)
}

var event = &definitions.Event{
	Id:         "93eb25a4-9348-4232-6f71-69e1e01081d7",
	Timestamp:  1499359162,
	Deployment: "loggregator",
	Message: &definitions.Event_Alert{
		Alert: &definitions.Alert{
			Severity: 4,
			Category: "",
			Title:    "SSH Access Denied",
			Summary:  "Failed password for vcap from 10.244.0.1 port 38732 ssh2",
			Source:   "loggregator: log-api(6f721317-2399-4e38-b38c-9d1b213c2d67) [id=130a69f5-6da1-45ce-830e-31e9c856085a, index=0, cid=b5df1c77-2c91-4093-6fc5-1cf2cba72471]",
		},
	},
}
