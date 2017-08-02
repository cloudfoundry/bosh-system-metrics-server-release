package egress_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestBoshMetricsWritesEventSuccessfully(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	sender := newSpyEgressSender(validContext("test-token"))
	tokenChecker := newSpyTokenChecker(nil)
	server := egress.NewServer(messages, tokenChecker)
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	go server.BoshMetrics(req, sender)

	messages <- event

	Eventually(sender.Received).Should(Receive(Equal(event)))
	Eventually(tokenChecker.received).Should(Receive(Equal("test-token")))
}

func TestBoshMetricsReturnsErrorWhenUnableToSend(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	sender := newSpyEgressSender(validContext("test-token"))
	sender.SendError = errors.New("unable to send")
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	var returnErr error
	done := make(chan struct{})
	go func() {
		returnErr = server.BoshMetrics(req, sender)
		close(done)
	}()
	messages <- event
	<-done

	Expect(sender.Received).To(BeEmpty())
	Expect(returnErr).ToNot(BeNil())
}

func TestBoshMetricsWithoutIncomingContext(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	invalidContext := context.Background()
	sender := newSpyEgressSender(invalidContext)
	sender.SendError = errors.New("unable to send")
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	err := server.BoshMetrics(req, sender)

	Expect(err).To(HaveOccurred())
}

func TestBoshMetricsWhenTokenIsInvalid(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	sender := newSpyEgressSender(validContext("test-token"))
	tokenChecker := newSpyTokenChecker(errors.New("token-invalid"))
	server := egress.NewServer(messages, tokenChecker)
	req := &definitions.EgressRequest{}

	err := server.BoshMetrics(req, sender)

	Expect(err).To(HaveOccurred())

	status, _ := status.FromError(err)
	Expect(status.Code()).To(Equal(codes.PermissionDenied))
}

func TestBoshMetricsWithoutAuthorizationHeader(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	contextWithNoHeader := metadata.NewIncomingContext(context.Background(), metadata.New(nil))
	sender := newSpyEgressSender(contextWithNoHeader)
	sender.SendError = errors.New("unable to send")
	server := egress.NewServer(messages, newSpyTokenChecker(nil))
	req := &definitions.EgressRequest{}

	err := server.BoshMetrics(req, sender)

	Expect(err).To(HaveOccurred())
}

// ------ SPIES ------
type spyEgressSender struct {
	Received  chan *definitions.Event
	SendError error
	grpc.ServerStream
	Token   string
	context context.Context
}

func newSpyEgressSender(ctx context.Context) *spyEgressSender {
	return &spyEgressSender{
		Received: make(chan *definitions.Event, 100),
		context:  ctx,
	}
}

func (s *spyEgressSender) Context() context.Context {
	return s.context
}

func validContext(token string) context.Context {
	md := metadata.New(map[string]string{
		"authorization": token,
	})

	return metadata.NewIncomingContext(context.Background(), md)
}

func (s *spyEgressSender) Send(e *definitions.Event) error {
	if s.SendError != nil {
		return s.SendError
	}

	s.Received <- e
	return nil
}

type spyTokenChecker struct {
	err      error
	received chan string
}

func newSpyTokenChecker(e error) *spyTokenChecker {
	return &spyTokenChecker{
		err:      e,
		received: make(chan string, 1),
	}
}

func (t *spyTokenChecker) CheckToken(token string) error {
	t.received <- token
	return t.err
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
