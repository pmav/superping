package ping

type PingResult struct {
	PingUUID  string `json:"ping_uuid"`
	Timestamp string `json:"timestamp"`
	Duration  int64  `json:"duration"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

const (
	ResultSuccess = "success"
	ResultFailed  = "failed"
)
