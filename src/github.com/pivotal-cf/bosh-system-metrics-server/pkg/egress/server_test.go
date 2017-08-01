package egress_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"golang.org/x/net/context"
)

func TestBoshMetricsWritesEventSuccessfully(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	sender := newSpyEgressSender()
	server := egress.NewServer(messages)
	req := &definitions.EgressRequest{SubscriptionId: "subscriptionA"}

	go server.BoshMetrics(req, sender)

	messages <- event

	Eventually(sender.Received).Should(Receive(Equal(event)))
}

func TestBoshMetricsReturnsErrorWhenUnableToSend(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	sender := newSpyEgressSender()
	sender.SendError = errors.New("unable to send")
	server := egress.NewServer(messages)
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

// ------ SPIES ------
type spyEgressSender struct {
	Received  chan *definitions.Event
	SendError error
	grpc.ServerStream
}

func newSpyEgressSender() *spyEgressSender {
	return &spyEgressSender{
		Received: make(chan *definitions.Event, 100),
	}
}

func (s *spyEgressSender) Context() context.Context {
	return context.Background()
}

func (s *spyEgressSender) Send(e *definitions.Event) error {
	if s.SendError != nil {
		return s.SendError
	}

	s.Received <- e
	return nil
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
