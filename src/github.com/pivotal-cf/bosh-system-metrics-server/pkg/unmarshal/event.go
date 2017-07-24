package unmarshal

import (
	"encoding/json"
	"errors"

	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
)

func Event(eventJSON []byte) (*definitions.Event, error) {
	var evt event

	err := json.Unmarshal(eventJSON, &evt)
	if err != nil {
		return nil, err
	}

	switch evt.Kind {
	case "heartbeat":
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
		Timestamp:  evt.CreatedAt,
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
			Timestamp: m.Timestamp,
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
		Timestamp:  evt.Timestamp,
		Message: &definitions.Event_Heartbeat{
			Heartbeat: &definitions.Heartbeat{
				AgentId:    evt.AgentId,
				Job:        evt.Job,
				Index:      evt.Index,
				InstanceId: evt.InstanceId,
				JobState:   evt.JobState,
				Vitals: &definitions.Heartbeat_Vitals{
					Cpu: &definitions.Heartbeat_Vitals_Cpu{
						Sys:  evt.Vitals.Cpu.Sys,
						User: evt.Vitals.Cpu.User,
						Wait: evt.Vitals.Cpu.Wait,
					},
					Disk: &definitions.Heartbeat_Vitals_Disk{
						Ephemeral: &definitions.Heartbeat_Vitals_Disk_DiskUsage{
							InodePercent: evt.Vitals.Disk.Ephemeral.InodePercent,
							Percent:      evt.Vitals.Disk.Ephemeral.Percent,
						},
						Persistent: &definitions.Heartbeat_Vitals_Disk_DiskUsage{
							InodePercent: evt.Vitals.Disk.Persistent.InodePercent,
							Percent:      evt.Vitals.Disk.Persistent.Percent,
						},
						System: &definitions.Heartbeat_Vitals_Disk_DiskUsage{
							InodePercent: evt.Vitals.Disk.System.InodePercent,
							Percent:      evt.Vitals.Disk.System.Percent,
						},
					},
					Load: loadValues,
					Mem: &definitions.Heartbeat_Vitals_MemUsage{
						Kb:      evt.Vitals.Mem.Kb,
						Percent: evt.Vitals.Mem.Percent,
					},
					Swap: &definitions.Heartbeat_Vitals_MemUsage{
						Kb:      evt.Vitals.Swap.Kb,
						Percent: evt.Vitals.Swap.Percent,
					},
				},
				Metrics: metrics,
			},
		},
	}
}
