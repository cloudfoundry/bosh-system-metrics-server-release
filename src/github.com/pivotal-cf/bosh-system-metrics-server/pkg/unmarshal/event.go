package unmarshal

import (
	"encoding/json"
	"errors"

	"time"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"log"
)

func Event(eventJSON []byte) (*definitions.Event, error) {
	var evt event

	err := json.Unmarshal(eventJSON, &evt)
	if err != nil {
		return nil, err
	}

	switch evt.Kind {
	case "heartbeat":
		env := mapHeartbeat(evt)
		log.Printf("JSON Event: %v\n", evt)
		log.Printf("System Metrics Envelope: %v\n", env)
		return mapHeartbeat(evt), nil
	case "alert":
		return mapAlert(evt), nil
	default:
		return nil, errors.New("event kind must be alert or heartbeat")
	}
}

func mapAlert(evt event) *definitions.Event {
	return &definitions.Event{
		Id:         evt.Id,
		Deployment: evt.Deployment,
		Timestamp:  time.Unix(evt.CreatedAt, 0).UnixNano(),
		Message: &definitions.Event_Alert{
			Alert: &definitions.Alert{
				Severity: evt.Severity,
				Category: evt.Category,
				Title:    evt.Title,
				Summary:  evt.Summary,
				Source:   evt.Source,
			},
		},
	}
}

func mapHeartbeat(evt event) *definitions.Event {
	metrics := make([]*definitions.Heartbeat_Metric, len(evt.Metrics))
	for i, m := range evt.Metrics {
		metrics[i] = &definitions.Heartbeat_Metric{
			Name:      m.Name,
			Value:     m.Value,
			Timestamp: time.Unix(m.Timestamp, 0).UnixNano(),
			Tags:      m.Tags,
		}
	}

	loadValues := make([]float32, len(evt.Vitals.Load))
	for i, l := range evt.Vitals.Load {
		loadValues[i] = float32(l)
	}

	return &definitions.Event{
		Id:         evt.Id,
		Deployment: evt.Deployment,
		Timestamp:  time.Unix(evt.Timestamp, 0).UnixNano(),
		Message: &definitions.Event_Heartbeat{
			Heartbeat: &definitions.Heartbeat{
				AgentId:    evt.AgentId,
				Job:        evt.Job,
				Index:      evt.Index,
				InstanceId: evt.InstanceId,
				JobState:   evt.JobState,
				Metrics:    metrics,
			},
		},
	}
}
