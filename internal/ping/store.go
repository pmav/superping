package ping

import (
	"context"
	"time"
)

type Store interface {
	Initialize(context.Context) error
	List(context.Context) ([]Ping, error)
	Add(context.Context, Ping) error
	Get(context.Context, string) (Ping, bool, error)
	Update(context.Context, string, string, string, string, string) (Ping, bool, error)
	Remove(context.Context, string) (bool, error)
	AddResult(context.Context, PingResult) error
	ListResults(context.Context, string) ([]PingResult, error)
	ListRecentResults(context.Context, string, int) ([]PingResult, error)
	ListRecentResultAggregations(context.Context, string, string, time.Time, int) ([]PingResultAggregation, error)
}
