package storage

import (
	"context"

	"github.com/example/superping/internal/ping"
)

func (store *Store) initializeResultTable(ctx context.Context) error {
	_, err := store.database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ping_result (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ping_uuid TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			duration INTEGER NOT NULL,
			status TEXT NOT NULL,
			message TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (ping_uuid) REFERENCES ping(uuid)
		)
	`)
	return err
}

func (store *Store) AddResult(ctx context.Context, result ping.PingResult) error {
	_, err := store.database.ExecContext(ctx, `INSERT INTO ping_result (ping_uuid, timestamp, duration, status, message) VALUES (?, ?, ?, ?, ?)`, result.PingUUID, result.Timestamp, result.Duration, result.Status, result.Message)
	return err
}

func (store *Store) ListResults(ctx context.Context, pingUUID string) ([]ping.PingResult, error) {
	return store.ListRecentResults(ctx, pingUUID, 1000)
}

func (store *Store) ListRecentResults(ctx context.Context, pingUUID string, limit int) ([]ping.PingResult, error) {
	if limit < 1 || limit > 1000 {
		limit = 30
	}
	rows, err := store.database.QueryContext(ctx, `SELECT ping_uuid, timestamp, duration, status, message FROM ping_result WHERE ping_uuid = ? ORDER BY id DESC LIMIT ?`, pingUUID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ping.PingResult, 0)
	for rows.Next() {
		var result ping.PingResult
		if err := rows.Scan(&result.PingUUID, &result.Timestamp, &result.Duration, &result.Status, &result.Message); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	for left, right := 0, len(results)-1; left < right; left, right = left+1, right-1 {
		results[left], results[right] = results[right], results[left]
	}
	return results, rows.Err()
}