package ingress_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
)

func TestEventProcessedUponSucessfulUnmarshal(t *testing.T) {
	RegisterTestingT(t)
	log.SetOutput(ioutil.Discard)

	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.on("success\n", event)
	reader := bytes.NewBufferString("success\n")
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(reader, fakeUnmarshaller.f, messages)

	defer ingestor.Start()()

	Eventually(messages).Should(Receive(Equal(event)))
}

func TestEventProcessingContinuesAfterReadError(t *testing.T) {
	RegisterTestingT(t)

	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.on("success\n", event)
	reader := newFakeErrorReader()
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(reader, fakeUnmarshaller.f, messages)

	defer ingestor.Start()()

	Consistently(messages).Should(BeEmpty())
}

func TestEventProcessingContinuesAfterUnmarshallError(t *testing.T) {
	RegisterTestingT(t)

	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.failOn("bad-json\n", errors.New("invalid json"))
	reader := bytes.NewBufferString("bad-json\n")
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(reader, fakeUnmarshaller.f, messages)

	defer ingestor.Start()()

	Consistently(messages).Should(BeEmpty())
}

func TestEventProcessingStopsAfterStoppingIngestor(t *testing.T) {
	RegisterTestingT(t)
	log.SetOutput(ioutil.Discard)

	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.on("success\n", event)
	reader := bytes.NewBufferString("success\n")
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(reader, fakeUnmarshaller.f, messages)

	stop := ingestor.Start()

	Eventually(messages).Should(HaveLen(1))
	<-messages
	stop()

	Consistently(messages).ShouldNot(Receive())
	Expect(messages).To(BeEmpty())
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

type fakeUnmarshaller struct {
	returnValues map[string]*definitions.Event
	errorValues  map[string]error
}

func newFakeUnmarshaller() *fakeUnmarshaller {
	return &fakeUnmarshaller{
		returnValues: make(map[string]*definitions.Event),
		errorValues:  make(map[string]error),
	}
}

func (f *fakeUnmarshaller) on(key string, evt *definitions.Event) {
	f.returnValues[key] = evt
}

func (f *fakeUnmarshaller) failOn(key string, err error) {
	f.errorValues[key] = err
}

func (f *fakeUnmarshaller) f(b []byte) (*definitions.Event, error) {
	evt, ok := f.returnValues[string(b)]
	if !ok {

		err, ok := f.errorValues[string(b)]
		if ok {
			return nil, err
		}

		return nil, errors.New(fmt.Sprintf("stub value: %s not found", string(b)))
	}

	return evt, nil
}

type fakeReader struct{}

func newFakeErrorReader() *fakeReader {
	return &fakeReader{}
}

func (r *fakeReader) ReadBytes(byte) ([]byte, error) {
	return nil, errors.New("read error")
}
