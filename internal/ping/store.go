package ping

import (
	"context"
	"database/sql"
)

type Store struct {
	database *sql.DB
}

func NewStore(database *sql.DB) *Store {
	return &Store{database: database}
}

func (store *Store) Initialize(ctx context.Context) error {
	_, err := store.database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ping (
			uuid TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			ip TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'off',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	rows, err := store.database.QueryContext(ctx, `PRAGMA table_info(ping)`)
	if err != nil {
		return err
	}
	columns := make(map[string]bool)
	for rows.Next() {
		var columnID int
		var columnName, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&columnID, &columnName, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		columns[columnName] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for columnName, definition := range map[string]string{
		"status":     `TEXT NOT NULL DEFAULT 'off'`,
		"created_at": `TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		"updated_at": `TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`,
	} {
		if !columns[columnName] {
			if _, err := store.database.ExecContext(ctx, "ALTER TABLE ping ADD COLUMN "+columnName+" "+definition); err != nil {
				return err
			}
		}
	}
	_, err = store.database.ExecContext(ctx, `
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

func (store *Store) List(ctx context.Context) ([]Ping, error) {
	rows, err := store.database.QueryContext(ctx, `
		SELECT p.uuid, p.name, p.ip, p.status, p.created_at, p.updated_at,
			COALESCE((SELECT COUNT(*) FROM ping_result pr WHERE pr.ping_uuid = p.uuid), 0)
		FROM ping p
		ORDER BY p.name, p.uuid
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pings := make([]Ping, 0)
	for rows.Next() {
		var item Ping
		if err := rows.Scan(&item.UUID, &item.Name, &item.IP, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.TotalResults); err != nil {
			return nil, err
		}
		pings = append(pings, item)
	}
	return pings, rows.Err()
}

func (store *Store) Add(ctx context.Context, item Ping) error {
	_, err := store.database.ExecContext(ctx, `INSERT INTO ping (uuid, name, ip, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, item.UUID, item.Name, item.IP, item.Status, item.CreatedAt, item.UpdatedAt)
	return err
}

func (store *Store) Get(ctx context.Context, uuid string) (Ping, bool, error) {
	var item Ping
	err := store.database.QueryRowContext(ctx, `
		SELECT p.uuid, p.name, p.ip, p.status, p.created_at, p.updated_at,
			COALESCE((SELECT COUNT(*) FROM ping_result pr WHERE pr.ping_uuid = p.uuid), 0)
		FROM ping p
		WHERE p.uuid = ?
	`, uuid).Scan(&item.UUID, &item.Name, &item.IP, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.TotalResults)
	if err == sql.ErrNoRows {
		return Ping{}, false, nil
	}
	if err != nil {
		return Ping{}, false, err
	}
	return item, true, nil
}

func (store *Store) Update(ctx context.Context, uuid, name, ip, status, updatedAt string) (Ping, bool, error) {
	result, err := store.database.ExecContext(ctx, `UPDATE ping SET name = ?, ip = ?, status = ?, updated_at = ? WHERE uuid = ?`, name, ip, status, updatedAt, uuid)
	if err != nil {
		return Ping{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Ping{}, false, err
	}
	if count == 0 {
		return Ping{}, false, nil
	}

	item, _, err := store.Get(ctx, uuid)
	return item, true, err
}

func (store *Store) AddResult(ctx context.Context, result PingResult) error {
	_, err := store.database.ExecContext(ctx, `INSERT INTO ping_result (ping_uuid, timestamp, duration, status, message) VALUES (?, ?, ?, ?, ?)`, result.PingUUID, result.Timestamp, result.Duration, result.Status, result.Message)
	return err
}

func (store *Store) ListResults(ctx context.Context, pingUUID string) ([]PingResult, error) {
	return store.ListRecentResults(ctx, pingUUID, 1000)
}

func (store *Store) ListRecentResults(ctx context.Context, pingUUID string, limit int) ([]PingResult, error) {
	if limit < 1 || limit > 1000 {
		limit = 30
	}
	rows, err := store.database.QueryContext(ctx, `SELECT ping_uuid, timestamp, duration, status, message FROM ping_result WHERE ping_uuid = ? ORDER BY id DESC LIMIT ?`, pingUUID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]PingResult, 0)
	for rows.Next() {
		var result PingResult
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

func (store *Store) Remove(ctx context.Context, uuid string) (bool, error) {
	result, err := store.database.ExecContext(ctx, `DELETE FROM ping WHERE uuid = ?`, uuid)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}
