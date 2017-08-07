package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"time"
	"fmt"
)

const writeDeadline = 2 * time.Second

func main() {

	serverPort := flag.Int("server-port", 25594, "The destination port to send events on localhost")
	flag.Parse()

	log.Printf("Starting system metrics plugin...")
	in := bufio.NewReader(os.Stdin)

	for {
		forwardMetricsToServer(in, *serverPort)
		time.Sleep(time.Second)
		log.Println("reconnecting to system metrics server...")
	}
}

func forwardMetricsToServer(in *bufio.Reader, port int) {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Printf("unable to connect to system metrics server: %s", err)
		return
	}
	log.Printf("connected to system metrics server at %s", conn.LocalAddr().String())

	for {
		b, err := in.ReadBytes('\n')
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		conn.SetWriteDeadline(time.Now().Add(writeDeadline))
		_, err = conn.Write(b)
		if err != nil {
			log.Printf("unable to write to system metrics server: %s", err)
			time.Sleep(time.Second)
			return
		}
	}
}
