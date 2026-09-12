package repository

import (
	"context"
	"database/sql"

	"igoblog/internal/apperr"
)

// settingsRepo 是 SettingsRepository 的 SQLite 实现。
type settingsRepo struct{ db *sql.DB }

// NewSettingsRepository 构造站点设置仓储。
func NewSettingsRepository(db *sql.DB) SettingsRepository { return &settingsRepo{db: db} }

func (r *settingsRepo) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, apperr.Internal(err)
		}
		settings[k] = v
	}
	return settings, rows.Err()
}

func (r *settingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}
