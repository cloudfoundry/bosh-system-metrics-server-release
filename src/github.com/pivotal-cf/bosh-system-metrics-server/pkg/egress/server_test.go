package egress_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
)

func TestServerReturnsErrorWhenUnableToSend(t *testing.T) {
	RegisterTestingT(t)

	messages := make(chan *definitions.Event, 100)
	writer := newSpyWriter()
	server := egress.NewServer(messages)
	go server.Start()

	messages <- event

	Expect(writer.Messages).To(HaveLen(1))
	Expect(writer.Messages).To(ContainElement(event))
}

type spyWriter struct {
	Messages []*definitions.Event
}

func newSpyWriter() *spyWriter {
	return &spyWriter{
		Messages: make([]*definitions.Event, 100),
	}
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
