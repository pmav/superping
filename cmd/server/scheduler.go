package main

import (
	"context"
	"log"
	"time"
)

type resultAggregator interface {
	AggregateResults(context.Context, time.Time, time.Time) error
}

func runScheduledTask(ctx context.Context, aggregator resultAggregator) {
	for {
		now := time.Now()
		nextRun := now.Truncate(time.Minute).Add(5 * time.Second)
		if !nextRun.After(now) {
			nextRun = nextRun.Add(time.Minute)
		}

		timer := time.NewTimer(time.Until(nextRun))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			log.Println("Executing")
			periodEnd := time.Now().UTC().Truncate(time.Minute)
			periodStart := periodEnd.Add(-time.Minute)
			if err := aggregator.AggregateResults(ctx, periodStart, periodEnd); err != nil {
				log.Printf("aggregate ping results: %v", err)
			}
		}
	}
}