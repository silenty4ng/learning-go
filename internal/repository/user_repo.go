package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
)

// userRepo 是 UserRepository 的 SQLite 实现。
type userRepo struct{ db *sql.DB }

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *sql.DB) UserRepository { return &userRepo{db: db} }

func (r *userRepo) Create(ctx context.Context, u *model.User) error {
	now := time.Now().UTC().Truncate(time.Second)
	if u.Role == "" {
		u.Role = model.RoleUser
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, nickname, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		u.Username, u.Nickname, u.PasswordHash, u.Role, now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("用户名 %q 已被占用", u.Username)
		}
		return apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return apperr.Internal(err)
	}
	u.ID, u.CreatedAt, u.UpdatedAt = id, now, now
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, username, nickname, password_hash, role, created_at, updated_at FROM users WHERE id = ?`, id))
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, username, nickname, password_hash, role, created_at, updated_at FROM users WHERE username = ?`, username))
}

func (r *userRepo) List(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, nickname, password_hash, role, created_at, updated_at
		 FROM users ORDER BY id LIMIT ? OFFSET ?`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Nickname, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *userRepo) UpdateRole(ctx context.Context, id int64, role string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, role, timeNow(), id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("用户")
	}
	return nil
}

// Delete 在单个事务内级联删除：用户的评论 -> 用户的文章（FK 级联其评论与标签绑定）-> 用户本身。
func (r *userRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return apperr.Internal(err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, q := range []string{
		`DELETE FROM comments WHERE user_id = ?`,
		`DELETE FROM articles WHERE author_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return apperr.Internal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *userRepo) CountAdmins(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE role = ?`, model.RoleAdmin).Scan(&n); err != nil {
		return 0, apperr.Internal(err)
	}
	return n, nil
}

func scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.Nickname, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("用户")
		}
		return nil, apperr.Internal(err)
	}
	return &u, nil
}

func (r *userRepo) UpdateNickname(ctx context.Context, id int64, nickname string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET nickname = ?, updated_at = ? WHERE id = ?`, nickname, timeNow(), id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("用户")
	}
	return nil
}

// isUniqueViolation 判断是否为唯一约束冲突（modernc.org/sqlite 的错误文案）。
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
