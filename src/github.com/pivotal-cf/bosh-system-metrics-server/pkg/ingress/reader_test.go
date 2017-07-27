package ingress_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
)

func TestStartProcessesMessages(t *testing.T) {
	RegisterTestingT(t)
	log.SetOutput(ioutil.Discard)

	port := 25596
	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.on("success\n", event)
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(port, fakeUnmarshaller.f, messages)

	defer ingestor.Start()()

	conn, err := net.Dial("tcp", "127.0.0.1:25596")
	Expect(err).ToNot(HaveOccurred())
	defer conn.Close()
	_, err = conn.Write([]byte("success\n"))
	Expect(err).ToNot(HaveOccurred())

	Eventually(messages).Should(Receive(Equal(event)))
}

func TestStartContinuesAfterUnmarshallError(t *testing.T) {
	RegisterTestingT(t)
	log.SetOutput(ioutil.Discard)

	port := 25597
	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.failOn("bad-json\n", errors.New("invalid json"))
	fakeUnmarshaller.on("success\n", event)
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(port, fakeUnmarshaller.f, messages)

	defer ingestor.Start()()

	conn, err := net.Dial("tcp", "127.0.0.1:25597")
	Expect(err).ToNot(HaveOccurred())
	defer conn.Close()
	_, err = conn.Write([]byte("bad-json\n"))
	Expect(err).ToNot(HaveOccurred())
	_, err = conn.Write([]byte("success\n"))
	Expect(err).ToNot(HaveOccurred())

	Eventually(messages).Should(Receive(Equal(event)))
}

func TestEventProcessingStopsAfterStoppingIngestor(t *testing.T) {
	RegisterTestingT(t)
	log.SetOutput(ioutil.Discard)

	port := 25596
	fakeUnmarshaller := newFakeUnmarshaller()
	fakeUnmarshaller.on("success\n", event)
	messages := make(chan *definitions.Event, 100)
	ingestor := ingress.New(port, fakeUnmarshaller.f, messages)

	stop := ingestor.Start()

	conn, err := net.Dial("tcp", "127.0.0.1:25596")
	Expect(err).ToNot(HaveOccurred())
	defer conn.Close()
	_, err = conn.Write([]byte("success\n"))
	Expect(err).ToNot(HaveOccurred())

	<-messages
	stop()

	_, err = conn.Write([]byte("success\n"))
	Expect(err).ToNot(HaveOccurred())

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

type fakeListener struct {
	closeCallCount  int64
	acceptCallCount int64
	conn            net.Conn
	err             error
}

func newFakeListener(c net.Conn, e error) *fakeListener {
	return &fakeListener{
		conn: c,
		err:  e,
	}
}

func (l *fakeListener) Accept() (net.Conn, error) {
	atomic.AddInt64(&l.acceptCallCount, 1)
	return l.conn, l.err
}

func (l *fakeListener) AcceptCallCount() int64 {
	return atomic.LoadInt64(&l.acceptCallCount)
}

func (l *fakeListener) Close() error {
	return nil
}
func (l *fakeListener) Addr() net.Addr {
	return nil
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

type spyReader struct {
	callCount int64
	bytes     []byte
	err       error
}

func newSpyReader(b []byte, e error) *spyReader {
	return &spyReader{
		bytes: b,
		err:   e,
	}
}

func (r *spyReader) ReadBytes(byte) ([]byte, error) {
	atomic.AddInt64(&r.callCount, 1)
	return r.bytes, r.err
}

func (r *spyReader) CallCount() int64 {
	return atomic.LoadInt64(&r.callCount)
}
