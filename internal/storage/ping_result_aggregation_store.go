package storage

import (
	"context"
	"errors"
	"time"

	"github.com/example/superping/internal/ping"
)

const resultAggregationTableSchema = `
	CREATE TABLE IF NOT EXISTS ping_result_aggregation (
		ping_uuid TEXT NOT NULL,
		aggregation_type TEXT NOT NULL,
		period_start TEXT NOT NULL,
		period_end TEXT NOT NULL,
		total_count INTEGER NOT NULL,
		success_count INTEGER NOT NULL,
		failure_count INTEGER NOT NULL,
		average_duration REAL NOT NULL,
		minimum_duration INTEGER NOT NULL,
		maximum_duration INTEGER NOT NULL,
		PRIMARY KEY (ping_uuid, aggregation_type, period_start, period_end),
		FOREIGN KEY (ping_uuid) REFERENCES ping(uuid)
	)
`

func (store *Store) initializeResultAggregationTable(ctx context.Context) error {
	_, err := store.database.ExecContext(ctx, resultAggregationTableSchema)
	return err
}

func (store *Store) AggregateResults(ctx context.Context, periodStart, periodEnd time.Time) error {
	if !periodStart.Before(periodEnd) {
		return errors.New("aggregation period start must be before period end")
	}

	const timestampLayout = "2006-01-02T15:04:05.000000000Z07:00"
	_, err := store.database.ExecContext(ctx, `
		INSERT OR REPLACE INTO ping_result_aggregation (
			ping_uuid, aggregation_type, period_start, period_end, total_count,
			success_count, failure_count, average_duration, minimum_duration, maximum_duration
		)
		SELECT ping_uuid, ?, ?, ?, COUNT(*),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
			COALESCE(AVG(CASE WHEN status = ? THEN duration END), 0),
			COALESCE(MIN(CASE WHEN status = ? THEN duration END), 0),
			COALESCE(MAX(CASE WHEN status = ? THEN duration END), 0)
		FROM ping_result
		WHERE timestamp >= ? AND timestamp < ?
		GROUP BY ping_uuid
	`, ping.AggregationTypeMinute, periodStart.Format(time.RFC3339Nano), periodEnd.Format(time.RFC3339Nano), ping.ResultSuccess, ping.ResultFailed, ping.ResultSuccess, ping.ResultSuccess, ping.ResultSuccess, periodStart.Format(timestampLayout), periodEnd.Format(timestampLayout))
	return err
}

func (store *Store) ListRecentResultAggregations(ctx context.Context, pingUUID, aggregationType string, since time.Time, limit int) ([]ping.PingResultAggregation, error) {
	if limit < 1 || limit > 60 {
		limit = 60
	}
	rows, err := store.database.QueryContext(ctx, `
		SELECT ping_uuid, aggregation_type, period_start, period_end, total_count,
			success_count, failure_count, average_duration, minimum_duration, maximum_duration
		FROM ping_result_aggregation
		WHERE ping_uuid = ? AND aggregation_type = ? AND period_start >= ?
		ORDER BY period_start DESC
		LIMIT ?
	`, pingUUID, aggregationType, since.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aggregations := make([]ping.PingResultAggregation, 0)
	for rows.Next() {
		var aggregation ping.PingResultAggregation
		if err := rows.Scan(&aggregation.PingUUID, &aggregation.AggregationType, &aggregation.PeriodStart, &aggregation.PeriodEnd, &aggregation.TotalCount, &aggregation.SuccessCount, &aggregation.FailureCount, &aggregation.AverageDuration, &aggregation.MinimumDuration, &aggregation.MaximumDuration); err != nil {
			return nil, err
		}
		aggregations = append(aggregations, aggregation)
	}
	for left, right := 0, len(aggregations)-1; left < right; left, right = left+1, right-1 {
		aggregations[left], aggregations[right] = aggregations[right], aggregations[left]
	}
	return aggregations, rows.Err()
}