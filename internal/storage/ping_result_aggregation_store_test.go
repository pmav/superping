package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/example/superping/internal/ping"
	_ "modernc.org/sqlite"
)

func TestAggregateResultsCreatesIdempotentMinuteBuckets(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { database.Close() })

	store := NewStore(database)
	ctx := context.Background()
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	for _, item := range []ping.Ping{
		{UUID: "ping-a", Name: "A", IP: "192.0.2.1", Status: ping.StatusOn, CreatedAt: "now", UpdatedAt: "now"},
		{UUID: "ping-b", Name: "B", IP: "192.0.2.2", Status: ping.StatusOn, CreatedAt: "now", UpdatedAt: "now"},
		{UUID: "ping-c", Name: "C", IP: "192.0.2.3", Status: ping.StatusOn, CreatedAt: "now", UpdatedAt: "now"},
	} {
		if err := store.Add(ctx, item); err != nil {
			t.Fatal(err)
		}
	}

	results := []ping.PingResult{
		{PingUUID: "ping-a", Timestamp: "2026-10-21T12:00:00.500Z", Duration: 12, Status: ping.ResultSuccess},
		{PingUUID: "ping-a", Timestamp: "2026-10-21T12:00:30Z", Duration: 2000, Status: ping.ResultFailed},
		{PingUUID: "ping-a", Timestamp: "2026-10-21T12:00:59.999999999Z", Duration: 24, Status: ping.ResultSuccess},
		{PingUUID: "ping-a", Timestamp: "2026-10-21T12:01:00Z", Duration: 100, Status: ping.ResultSuccess},
		{PingUUID: "ping-b", Timestamp: "2026-10-21T12:00:45Z", Duration: 8, Status: ping.ResultSuccess},
		{PingUUID: "ping-c", Timestamp: "2026-10-21T12:00:45Z", Duration: 1000, Status: ping.ResultFailed},
	}
	for _, result := range results {
		if err := store.AddResult(ctx, result); err != nil {
			t.Fatal(err)
		}
	}

	periodStart := time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(time.Minute)
	if err := store.AggregateResults(ctx, periodStart, periodEnd); err != nil {
		t.Fatal(err)
	}

	var aggregationType, storedStart, storedEnd string
	var totalCount, successCount, failureCount, minimumDuration, maximumDuration int
	var averageDuration float64
	err = database.QueryRowContext(ctx, `
		SELECT aggregation_type, period_start, period_end, total_count, success_count,
			failure_count, average_duration, minimum_duration, maximum_duration
		FROM ping_result_aggregation WHERE ping_uuid = 'ping-a'
	`).Scan(&aggregationType, &storedStart, &storedEnd, &totalCount, &successCount, &failureCount, &averageDuration, &minimumDuration, &maximumDuration)
	if err != nil {
		t.Fatal(err)
	}
	if aggregationType != "1m" || storedStart != periodStart.Format(time.RFC3339Nano) || storedEnd != periodEnd.Format(time.RFC3339Nano) || totalCount != 3 || successCount != 2 || failureCount != 1 || averageDuration != 18 || minimumDuration != 12 || maximumDuration != 24 {
		t.Fatalf("aggregation = type %q period %q-%q counts %d/%d/%d average %.2f range %d-%d", aggregationType, storedStart, storedEnd, totalCount, successCount, failureCount, averageDuration, minimumDuration, maximumDuration)
	}

	var failedOnlySuccesses, failedOnlyFailures, failedOnlyMinimum, failedOnlyMaximum int
	var failedOnlyAverage float64
	err = database.QueryRowContext(ctx, `
		SELECT success_count, failure_count, average_duration, minimum_duration, maximum_duration
		FROM ping_result_aggregation WHERE ping_uuid = 'ping-c'
	`).Scan(&failedOnlySuccesses, &failedOnlyFailures, &failedOnlyAverage, &failedOnlyMinimum, &failedOnlyMaximum)
	if err != nil {
		t.Fatal(err)
	}
	if failedOnlySuccesses != 0 || failedOnlyFailures != 1 || failedOnlyAverage != 0 || failedOnlyMinimum != 0 || failedOnlyMaximum != 0 {
		t.Fatalf("failed-only aggregation = successes %d failures %d average %.2f range %d-%d", failedOnlySuccesses, failedOnlyFailures, failedOnlyAverage, failedOnlyMinimum, failedOnlyMaximum)
	}

	if err := store.AggregateResults(ctx, periodStart, periodEnd); err != nil {
		t.Fatal(err)
	}
	var aggregationRows int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM ping_result_aggregation`).Scan(&aggregationRows); err != nil {
		t.Fatal(err)
	}
	if aggregationRows != 3 {
		t.Fatalf("aggregation rows = %d, want one row per ping", aggregationRows)
	}
}