package storage

import (
	"context"
	"database/sql"

	"github.com/example/superping/internal/ping"
)

type Store struct {
	database *sql.DB
}

func NewStore(database *sql.DB) *Store {
	return &Store{database: database}
}

func (store *Store) initializePingTable(ctx context.Context) error {
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
	return nil
}

func (store *Store) List(ctx context.Context) ([]ping.Ping, error) {
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

	pings := make([]ping.Ping, 0)
	for rows.Next() {
		var item ping.Ping
		if err := rows.Scan(&item.UUID, &item.Name, &item.IP, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.TotalResults); err != nil {
			return nil, err
		}
		pings = append(pings, item)
	}
	return pings, rows.Err()
}

func (store *Store) Add(ctx context.Context, item ping.Ping) error {
	_, err := store.database.ExecContext(ctx, `INSERT INTO ping (uuid, name, ip, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, item.UUID, item.Name, item.IP, item.Status, item.CreatedAt, item.UpdatedAt)
	return err
}

func (store *Store) Get(ctx context.Context, uuid string) (ping.Ping, bool, error) {
	var item ping.Ping
	err := store.database.QueryRowContext(ctx, `
		SELECT p.uuid, p.name, p.ip, p.status, p.created_at, p.updated_at,
			COALESCE((SELECT COUNT(*) FROM ping_result pr WHERE pr.ping_uuid = p.uuid), 0)
		FROM ping p
		WHERE p.uuid = ?
	`, uuid).Scan(&item.UUID, &item.Name, &item.IP, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.TotalResults)
	if err == sql.ErrNoRows {
		return ping.Ping{}, false, nil
	}
	if err != nil {
		return ping.Ping{}, false, err
	}
	return item, true, nil
}

func (store *Store) Update(ctx context.Context, uuid, name, ip, status, updatedAt string) (ping.Ping, bool, error) {
	result, err := store.database.ExecContext(ctx, `UPDATE ping SET name = ?, ip = ?, status = ?, updated_at = ? WHERE uuid = ?`, name, ip, status, updatedAt, uuid)
	if err != nil {
		return ping.Ping{}, false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ping.Ping{}, false, err
	}
	if count == 0 {
		return ping.Ping{}, false, nil
	}

	item, _, err := store.Get(ctx, uuid)
	return item, true, err
}

func (store *Store) Remove(ctx context.Context, uuid string) (bool, error) {
	result, err := store.database.ExecContext(ctx, `DELETE FROM ping WHERE uuid = ?`, uuid)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count > 0, err
}
