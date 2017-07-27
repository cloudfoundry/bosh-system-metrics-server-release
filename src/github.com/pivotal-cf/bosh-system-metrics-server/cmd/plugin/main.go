package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"time"
)

const writeDeadline = 2 * time.Second

func main() {

	addr := flag.String("addr", "127.0.0.1:25594", "The destination address to send events")
	flag.Parse()

	log.Printf("Starting system metrics plugin...")
	in := bufio.NewReader(os.Stdin)

	for {
		forwardMetricsToServer(in, *addr)
		time.Sleep(time.Second)
	}
}

func forwardMetricsToServer(in *bufio.Reader, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Unable to connect to system metrics server: %s", err)
		return
	}

	for {
		b, err := in.ReadBytes('\n')
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		conn.SetWriteDeadline(time.Now().Add(writeDeadline))
		_, err = conn.Write(b)
		if err != nil {
			log.Printf("Unable to write to system metrics server, reconnecting: %s", err)
			time.Sleep(time.Second)
			return
		}
	}
}
