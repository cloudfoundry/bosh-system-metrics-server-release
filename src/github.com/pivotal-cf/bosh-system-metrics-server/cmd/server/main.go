package main

import (
	"bufio"
	"math/rand"
	"os"
	"time"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/egress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	messages := make(chan *definitions.Event)

	i := ingress.New(bufio.NewReader(os.Stdin), unmarshal.Event, messages)
	e := egress.NewServer(messages)

	stopReadingMessages := i.Start()
	stopWritingMessages := e.Start()

	defer func() {
		stopReadingMessages()
		stopWritingMessages()
	}()
}
