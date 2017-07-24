package unmarshal

import "encoding/json"

type event struct {
	Id         string
	Deployment string
	Kind       string

	// heartbeat
	Timestamp  int64     `json:"timestamp,omitempty"`
	AgentId    string    `json:"agent_id,omitempty"`
	Job        string    `json:"job,omitempty"`
	Index      int32     `json:"index,string,omitempty"`
	InstanceId string    `json:"instance_id,omitempty"`
	JobState   string    `json:"job_state,omitempty"`
	Vitals     *vitals   `json:"vitals,omitempty"`
	Metrics    []*metric `json:"metrics,omitempty"`

	// alert
	CreatedAt int64  `json:"created_at,omitempty"`
	Severity  int32  `json:"severity,omitempty"`
	Category  string `json:"category,omitempty"`
	Title     string `json:"title,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Source    string `json:"source,omitempty"`
}

type vitals struct {
	Cpu  *cpu
	Disk *disk
	Load []float32str
	Mem  *memUsage
	Swap *memUsage
}

type float32str float32

func (i *float32str) UnmarshalJSON(p []byte) error {
	var s string
	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), (*float32)(i))
}

type cpu struct {
	Sys  float32 `json:"sys,string"`
	User float32 `json:"user,string"`
	Wait float32 `json:"wait,string"`
}

type disk struct {
	Ephemeral  *diskUsage
	Persistent *diskUsage
	System     *diskUsage
}

type diskUsage struct {
	InodePercent float32 `json:"inode_percent,string"`
	Percent      float32 `json:"percent,string"`
}

type memUsage struct {
	Kb      int64   `json:",string"`
	Percent float32 `json:",string"`
}

type metric struct {
	Name      string
	Value     float64 `json:",string"`
	Timestamp int64
	Tags      map[string]string
}
