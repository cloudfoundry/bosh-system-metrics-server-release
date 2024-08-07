package unmarshal

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/cloudfoundry/bosh-system-metrics-server/pkg/definitions"
)

// Event unmarshalls the json bosh event into
// either a Heartbeat or Alert `definitions.Event`.
// It returns an error if the event is not one of the two mentioned.
func Event(eventJSON []byte) (*definitions.Event, error) {
	var evt event

	err := json.Unmarshal(eventJSON, &evt)
	if err != nil {
		return nil, fmt.Errorf("%s (%s)", err, string(eventJSON))
	}

	switch evt.Kind {
	case "heartbeat":
		heartbeat, err := mapHeartbeat(evt)
		if err != nil {
			return nil, err
		}
		return heartbeat, nil
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

func mapHeartbeat(evt event) (*definitions.Event, error) {
	index, err := getIndexFromString(evt)
	if err != nil {
		return nil, err
	}

	return &definitions.Event{
		Id:         evt.Id,
		Deployment: evt.Deployment,
		Timestamp:  time.Unix(evt.Timestamp, 0).UnixNano(),
		Message: &definitions.Event_Heartbeat{
			Heartbeat: &definitions.Heartbeat{
				AgentId:    evt.AgentId,
				Job:        evt.Job,
				Index:      *index,
				InstanceId: evt.InstanceId,
				JobState:   evt.JobState,
				Metrics:    filterMetricsWithValues(evt),
			},
		},
	}, nil
}

func getIndexFromString(evt event) (*int32, error) {
	if evt.Index == "" {
		zeroIndex := int32(0)
		return &zeroIndex, nil
	}

	index, err := strconv.Atoi(evt.Index)
	if err != nil {
		return nil, err
	}

	if index > math.MaxInt32 {
		integerOverflowError := fmt.Sprintf("integer overflow detected for casting index %d to int32", index)
		return nil, errors.New(integerOverflowError)
	}

	int32Index := int32(index) // #nosec G109 - Checked for integer overflow above
	return &int32Index, nil
}

func filterMetricsWithValues(evt event) []*definitions.Heartbeat_Metric {
	metrics := make([]*definitions.Heartbeat_Metric, 0)
	for _, m := range evt.Metrics {
		val, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			continue
		}

		metrics = append(metrics, &definitions.Heartbeat_Metric{
			Name:      m.Name,
			Value:     val,
			Timestamp: time.Unix(m.Timestamp, 0).UnixNano(),
			Tags:      m.Tags,
		})
	}
	return metrics
}
