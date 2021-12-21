package unmarshal

type event struct {
	Id         string  `json:"id"`
	Deployment string  `json:"deployment"`
	Kind       string  `json:"kind"`

	// heartbeat
	Timestamp  int64     `json:"timestamp,omitempty"`
	AgentId    string    `json:"agent_id,omitempty"`
	Job        string    `json:"job,omitempty"`
	Index      string    `json:"index,omitempty"`
	InstanceId string    `json:"instance_id,omitempty"`
	JobState   string    `json:"job_state,omitempty"`
	Metrics    []*metric `json:"metrics,omitempty"`

	// alert
	CreatedAt int64  `json:"created_at,omitempty"`
	Severity  int32  `json:"severity,omitempty"`
	Category  string `json:"category,omitempty"`
	Title     string `json:"title,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Source    string `json:"source,omitempty"`
}

type metric struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Timestamp int64 `json:"timestamp"`
	Tags      map[string]string
}
