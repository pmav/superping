package monitor

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/example/superping/internal/ping"
)

type Monitor struct {
	store  *ping.Store
	runner func(context.Context, string) error
}

func New(store *ping.Store) *Monitor {
	return &Monitor{store: store, runner: runPing}
}

func (monitor *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			monitor.checkAll(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (monitor *Monitor) checkAll(ctx context.Context) {
	pings, err := monitor.store.List(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("monitor list error=%v", err)
		}
		return
	}

	for _, item := range pings {
		if ctx.Err() != nil {
			return
		}
		if item.Status != ping.StatusOn {
			continue
		}
		pingContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		start := time.Now()
		err := monitor.runner(pingContext, item.IP)
		duration := time.Since(start)
		cancel()
		result := ping.PingResult{
			PingUUID:  item.UUID,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Duration:  duration.Milliseconds(),
			Status:    ping.ResultSuccess,
		}
		if err != nil {
			result.Status = ping.ResultFailed
			result.Message = err.Error()
			log.Printf("ping uuid=%s name=%q ip=%s status=down duration=%s error=%v", item.UUID, item.Name, item.IP, duration.Round(time.Millisecond), err)
		} else {
			log.Printf("ping uuid=%s name=%q ip=%s status=up duration=%s", item.UUID, item.Name, item.IP, duration.Round(time.Millisecond))
		}
		if err := monitor.store.AddResult(ctx, result); err != nil && ctx.Err() == nil {
			log.Printf("monitor result error uuid=%s error=%v", item.UUID, err)
		}
	}
}

func runPing(ctx context.Context, address string) error {
	command := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", address)
	if output, err := command.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ping command failed: %w (%s)", err, output)
	}
	return nil
}
