package repository

import (
	"context"
	"database/sql"
	"errors"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
)

// categoryRepo 是 CategoryRepository 的 SQLite 实现。
type categoryRepo struct{ db *sql.DB }

// NewCategoryRepository 构造分类仓储。
func NewCategoryRepository(db *sql.DB) CategoryRepository { return &categoryRepo{db: db} }

func (r *categoryRepo) Create(ctx context.Context, c *model.Category) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO categories (name) VALUES (?)`, c.Name)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("分类 %q 已存在", c.Name)
		}
		return apperr.Internal(err)
	}
	if c.ID, err = res.LastInsertId(); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *categoryRepo) GetByID(ctx context.Context, id int64) (*model.Category, error) {
	var c model.Category
	err := r.db.QueryRowContext(ctx, `SELECT id, name FROM categories WHERE id = ?`, id).
		Scan(&c.ID, &c.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("分类")
		}
		return nil, apperr.Internal(err)
	}
	return &c, nil
}

func (r *categoryRepo) List(ctx context.Context) ([]model.Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM categories ORDER BY id`)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	categories := []model.Category{}
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, apperr.Internal(err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *categoryRepo) Update(ctx context.Context, c *model.Category) error {
	res, err := r.db.ExecContext(ctx, `UPDATE categories SET name = ? WHERE id = ?`, c.Name, c.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("分类 %q 已存在", c.Name)
		}
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("分类")
	}
	return nil
}

func (r *categoryRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("分类")
	}
	return nil
}

func (r *categoryRepo) HasArticles(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM articles WHERE category_id = ?`, id).Scan(&n); err != nil {
		return false, apperr.Internal(err)
	}
	return n > 0, nil
}
