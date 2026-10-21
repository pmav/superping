package ping

type PingResult struct {
	PingUUID  string `json:"ping_uuid"`
	Timestamp string `json:"timestamp"`
	Duration  int64  `json:"duration"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

type PingResultAggregation struct {
	PingUUID        string  `json:"ping_uuid"`
	AggregationType string  `json:"aggregation_type"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
	TotalCount      int64   `json:"total_count"`
	SuccessCount    int64   `json:"success_count"`
	FailureCount    int64   `json:"failure_count"`
	AverageDuration float64 `json:"average_duration"`
	MinimumDuration int64   `json:"minimum_duration"`
	MaximumDuration int64   `json:"maximum_duration"`
}

const (
	ResultSuccess         = "success"
	ResultFailed          = "failed"
	AggregationTypeMinute = "1m"
)
