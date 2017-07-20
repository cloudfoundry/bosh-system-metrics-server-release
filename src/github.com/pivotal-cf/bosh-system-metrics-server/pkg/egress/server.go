package egress

import (
	"log"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
)

type eventWriter interface {
	Write(definitions.Egress_BoshMetricsServer, *definitions.Event) error
}

type BoshMetricsServer struct {
	// registry map[string][]chan *definitions.Event
	messages chan *definitions.Event
	writer   eventWriter
}

func NewServer(m chan *definitions.Event, w eventWriter) *BoshMetricsServer {
	return &BoshMetricsServer{
		// registry: make(map[string][]chan *definitions.Event),
		messages: m,
		writer:   w,
	}
}

func (s *BoshMetricsServer) Start() func() {
	// done := make(chan struct{})

	// go func() {
	// 	for event := range s.messages {
	// 		s.mu.RLock()
	// 		for _, buffers := range s.registry {
	// 			randomIdx := rand.Intn(len(buffers))
	// 			buffers[randomIdx] <- event
	// 		}
	// 		s.mu.RUnlock()
	// 	}
	// 	close(done)
	// }()

	// return func() {
	// 	s.mu.Lock()
	// 	defer s.mu.Unlock()

	// 	for _, buffers := range s.registry {
	// 		for _, buffer := range buffers {
	// 			close(buffer)
	// 		}
	// 	}

	// 	<-done
	// }
	return func() {}
}

func (s *BoshMetricsServer) BoshMetrics(r *definitions.EgressRequest, srv definitions.Egress_BoshMetricsServer) error {
	// buffer := make(chan *definitions.Event)

	// s.register(r.subscriptionId, buffer)
	// defer s.unregister(buffer)

	for event := range s.messages {
		err := s.writer.Write(srv, event)
		if err != nil {
			log.Printf("Send Error: %s", err)
			return err
		}
	}

	return nil
}

// func (s *BoshMetricsServer) register(subscriptionId string, buffer chan *definitions.Event) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	channels, ok := s.registry[subscriptionId]
// 	if !ok {
// 		channels := make([]chan *definitions.Event)
// 		s.registry[channels] = channels
// 	}

// 	s.registry[subscriptionId] = append(channels, buffer)
// }

// func (s *BoshMetricsServer) unregister(subscriptionId string, buffer chan *definitions.Event) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	channels, ok := s.registry[subscriptionId]
// 	for i, _ := range channels {
// 		if channels[i] == buffer {
// 			s.registry[subscriptionId] = append(channels[:i], channels[i+1:]...)
// 			return
// 		}
// 	}

// }
