package unmarshal_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/definitions"
	"github.com/pivotal-cf/bosh-system-metrics-server/pkg/unmarshal"
)

func TestHeartbeatConversion(t *testing.T) {
	RegisterTestingT(t)

	var heartbeatJSON = []byte(`
    {
       "kind":"heartbeat",
       "id":"55b68400-f984-4f76-b341-cf849e07d4f9",
       "timestamp":1499293724,
       "deployment":"loggregator",
       "agent_id":"2accd102-37e7-4dd6-b337-b3f87da97914",
       "job":"consul",
       "index":"4",
       "instance_id":"6f60a3ce-9e4d-477f-ba45-7d29bcfab5b9",
       "job_state":"running",
       "vitals":{
          "cpu":{
             "sys":"3.2",
             "user":"2.5",
             "wait":"0.0"
          },
          "disk":{
             "ephemeral":{
                "inode_percent":"2",
                "percent":"4"
             },
             "persistent":{
                "inode_percent":"2",
                "percent":"4"
             },
             "system":{
                "inode_percent":"14",
                "percent":"23"
             }
          },
          "load":[
             "0.18",
             "0.23",
             "0.29"
          ],
          "mem":{
             "kb":"1139140",
             "percent":"28"
          },
          "swap":{
             "kb":"9788",
             "percent":"2"
          }
       },
       "teams":[],
       "metrics":[
          {
             "name":"system.load.1m",
             "value":"2.5",
             "timestamp":1499293724,
             "tags":{
                "job":"consul",
                "index":"1",
                "id":"6f60a3ce-9e4d-477f-ba45-7d29bcfab5b9"
             }
          }
       ]
    }
    `)

	heartbeat, err := unmarshal.Event(heartbeatJSON)

	Expect(err).ToNot(HaveOccurred())

	Expect(heartbeat).To(Equal(&definitions.Event{
		Id:         "55b68400-f984-4f76-b341-cf849e07d4f9",
		Timestamp:  1499293724000000000,
		Deployment: "loggregator",
		Message: &definitions.Event_Heartbeat{
			Heartbeat: &definitions.Heartbeat{
				AgentId:    "2accd102-37e7-4dd6-b337-b3f87da97914",
				Job:        "consul",
				Index:      4,
				InstanceId: "6f60a3ce-9e4d-477f-ba45-7d29bcfab5b9",
				JobState:   "running",
				Metrics: []*definitions.Heartbeat_Metric{
					{
						Name:      "system.load.1m",
						Value:     2.5,
						Timestamp: 1499293724000000000,
						Tags: map[string]string{
							"job":   "consul",
							"index": "1",
							"id":    "6f60a3ce-9e4d-477f-ba45-7d29bcfab5b9",
						},
					},
				},
			},
		},
	}))
}

func TestAlertConversion(t *testing.T) {
	RegisterTestingT(t)

	var alertJSON = []byte(`
		{
           "kind":"alert",
           "id":"93eb25a4-9348-4232-6f71-69e1e01081d7",
           "severity":4,
           "category":null,
           "title":"SSH Access Denied",
           "summary":"Failed password for vcap from 10.244.0.1 port 38732 ssh2",
           "source":"loggregator: log-api(6f721317-2399-4e38-b38c-9d1b213c2d67) [id=130a69f5-6da1-45ce-830e-31e9c856085a, index=0, cid=b5df1c77-2c91-4093-6fc5-1cf2cba72471]",
           "deployment":"loggregator",
           "created_at":1499359162
        }`)

	alert, err := unmarshal.Event(alertJSON)
	Expect(err).ToNot(HaveOccurred())

	Expect(alert).To(Equal(&definitions.Event{
		Id:         "93eb25a4-9348-4232-6f71-69e1e01081d7",
		Timestamp:  1499359162000000000,
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
	}))
}

func TestInvalidEvent(t *testing.T) {
	RegisterTestingT(t)

	var heartbeatJSON = []byte(` { } `)

	heartbeat, err := unmarshal.Event(heartbeatJSON)

	Expect(heartbeat).To(BeNil())
	Expect(err).To(HaveOccurred())
}
