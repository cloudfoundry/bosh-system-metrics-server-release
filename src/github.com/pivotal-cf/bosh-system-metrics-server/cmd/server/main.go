package main

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/ingress"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
)

func main() {
	input := readFromStdIn()
	rand.Seed(time.Now().UnixNano())

	i, messages := ingress.New(input, unmarshal.Event)
	e := egress.New(messages)

	stopReadingMessages := i.Start()
	stopWritingMessages := e.Start()

	defer func() {
		stopReadingMessages()
		stopWritingMessages()
	}()
}

func readFromStdIn() chan []byte {
	input := make(chan []byte)

	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			line, err := r.ReadBytes('\n')
			if err != nil {
				log.Printf("Error reading from stdin: %s", err)
				continue
			}

			input <- line
			select {
			case input <- line:
			default:
				log.Printf("dropped stdin json metric, not processing fast enough")
			}
		}
	}()

	return input
}
