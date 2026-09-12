package repository

import (
	"context"
	"database/sql"
	"errors"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
)

// commentRepo 是 CommentRepository 的 SQLite 实现。
type commentRepo struct{ db *sql.DB }

// NewCommentRepository 构造评论仓储。
func NewCommentRepository(db *sql.DB) CommentRepository { return &commentRepo{db: db} }

func (r *commentRepo) Create(ctx context.Context, c *model.Comment) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (article_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
		c.ArticleID, c.UserID, c.Content, timeNow())
	if err != nil {
		return apperr.Internal(err)
	}
	if c.ID, err = res.LastInsertId(); err != nil {
		return apperr.Internal(err)
	}
	c.CreatedAt = timeNow()
	return nil
}

func (r *commentRepo) GetByID(ctx context.Context, id int64) (*model.Comment, error) {
	var c model.Comment
	err := r.db.QueryRowContext(ctx,
		`SELECT c.id, c.article_id, c.user_id, COALESCE(NULLIF(u.nickname, ''), u.username), c.content, c.created_at
		 FROM comments c JOIN users u ON u.id = c.user_id WHERE c.id = ?`, id).
		Scan(&c.ID, &c.ArticleID, &c.UserID, &c.Username, &c.Content, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("评论")
		}
		return nil, apperr.Internal(err)
	}
	return &c, nil
}

func (r *commentRepo) ListByArticle(ctx context.Context, articleID int64, page, pageSize int) ([]model.Comment, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM comments WHERE article_id = ?`, articleID).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.article_id, c.user_id, COALESCE(NULLIF(u.nickname, ''), u.username), c.content, c.created_at
		 FROM comments c JOIN users u ON u.id = c.user_id
		 WHERE c.article_id = ?
		 ORDER BY c.created_at ASC, c.id ASC
		 LIMIT ? OFFSET ?`, articleID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()

	comments := []model.Comment{}
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.ArticleID, &c.UserID, &c.Username, &c.Content, &c.CreatedAt); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *commentRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("评论")
	}
	return nil
}
