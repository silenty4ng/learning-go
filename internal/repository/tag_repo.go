package repository

import (
	"context"
	"database/sql"
	"errors"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
)

// tagRepo 是 TagRepository 的 SQLite 实现。
type tagRepo struct{ db *sql.DB }

// NewTagRepository 构造标签仓储。
func NewTagRepository(db *sql.DB) TagRepository { return &tagRepo{db: db} }

func (r *tagRepo) Create(ctx context.Context, t *model.Tag) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?)`, t.Name)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("标签 %q 已存在", t.Name)
		}
		return apperr.Internal(err)
	}
	if t.ID, err = res.LastInsertId(); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *tagRepo) GetByID(ctx context.Context, id int64) (*model.Tag, error) {
	var t model.Tag
	err := r.db.QueryRowContext(ctx, `SELECT id, name FROM tags WHERE id = ?`, id).
		Scan(&t.ID, &t.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("标签")
		}
		return nil, apperr.Internal(err)
	}
	return &t, nil
}

func (r *tagRepo) List(ctx context.Context) ([]model.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM tags ORDER BY id`)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	tags := []model.Tag{}
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, apperr.Internal(err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *tagRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("标签")
	}
	return nil
}

// SetArticleTags 全量重设文章-标签绑定：先删后插，包在一个事务里保证原子性。
func (r *tagRepo) SetArticleTags(ctx context.Context, articleID int64, tagIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return apperr.Internal(err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM article_tags WHERE article_id = ?`, articleID); err != nil {
		return apperr.Internal(err)
	}
	if len(tagIDs) > 0 {
		stmt, err := tx.PrepareContext(ctx,
			`INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)`)
		if err != nil {
			return apperr.Internal(err)
		}
		defer stmt.Close()
		for _, id := range tagIDs {
			if _, err := stmt.ExecContext(ctx, articleID, id); err != nil {
				return apperr.Internal(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (r *tagRepo) ListByArticle(ctx context.Context, articleID int64) ([]model.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.name FROM tags t
		 JOIN article_tags at ON at.tag_id = t.id
		 WHERE at.article_id = ? ORDER BY t.id`, articleID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()

	tags := []model.Tag{}
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, apperr.Internal(err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
