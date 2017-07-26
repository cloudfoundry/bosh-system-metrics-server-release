package ingress

import (
	"io"
	"log"
	"time"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
)

type unmarshaller func(eventJSON []byte) (*definitions.Event, error)

type reader interface {
	ReadBytes(delim byte) ([]byte, error)
}

type Ingestor struct {
	reader       reader
	unmarshaller unmarshaller
	output       chan *definitions.Event
}

func New(r reader, u unmarshaller, m chan *definitions.Event) *Ingestor {
	return &Ingestor{
		reader:       r,
		unmarshaller: u,
		output:       m,
	}
}

func (i *Ingestor) Start() func() {

	stop := make(chan struct{})

	go func() {
		for {
			b, err := i.reader.ReadBytes('\n')
			if err == io.EOF {
				time.Sleep(time.Second)
				continue
			}
			if err != nil {
				log.Printf("Error reading: %s", err)
				continue
			}

			evt, err := i.unmarshaller(b)
			if err != nil {
				log.Printf("Error unmarshalling: %s", err)
				continue
			}

			select {
			case <-stop:
				return
			case i.output <- evt:
			}
		}
	}()

	return func() {
		close(stop)
	}
}
