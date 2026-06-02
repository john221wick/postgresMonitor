package agentserver

type TopologyResponse struct {
	OK bool `json:"ok"`
}

type JobSpec struct {
	ID      string `json:"id"`
	Command string `json:"command"`
}

type JobStatusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type StatusResponse struct {
	Jobs []JobStatusResponse `json:"jobs"`
}

type LogChunk struct {
	Data   string `json:"data"`
	Offset int64  `json:"offset"`
	EOF    bool   `json:"eof"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}