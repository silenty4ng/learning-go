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

// articleRepo 是 ArticleRepository 的 SQLite 实现。
type articleRepo struct{ db *sql.DB }

// NewArticleRepository 构造文章仓储。
func NewArticleRepository(db *sql.DB) ArticleRepository { return &articleRepo{db: db} }

// 联表查询的公共片段：作者名与分类名一并取出，避免 handler 层二次查询。
const (
	// author 展示昵称优先：COALESCE(NULLIF(nickname,''), username)
	articleCols = `a.id, a.title, a.content, a.author_id, COALESCE(NULLIF(u.nickname, ''), u.username) AS author,
		a.category_id, c.name AS category, a.view_count, a.created_at, a.updated_at`
	articleFrom = ` FROM articles a
		JOIN users u ON u.id = a.author_id
		JOIN categories c ON c.id = a.category_id`
)

func (r *articleRepo) Create(ctx context.Context, a *model.Article) error {
	now := time.Now().UTC().Truncate(time.Second)
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO articles (title, content, author_id, category_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.Title, a.Content, a.AuthorID, a.CategoryID, now, now)
	if err != nil {
		return apperr.Internal(err)
	}
	if a.ID, err = res.LastInsertId(); err != nil {
		return apperr.Internal(err)
	}
	a.ViewCount, a.CreatedAt, a.UpdatedAt = 0, now, now
	return nil
}

func (r *articleRepo) GetByID(ctx context.Context, id int64) (*model.Article, error) {
	a, err := scanArticle(r.db.QueryRowContext(ctx,
		`SELECT `+articleCols+articleFrom+` WHERE a.id = ?`, id))
	if err != nil {
		return nil, err
	}
	// 附带标签列表
	tags, err := r.listTags(ctx, id)
	if err != nil {
		return nil, err
	}
	a.Tags = tags
	return a, nil
}

func (r *articleRepo) Update(ctx context.Context, a *model.Article) error {
	a.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	res, err := r.db.ExecContext(ctx,
		`UPDATE articles SET title = ?, content = ?, category_id = ?, updated_at = ? WHERE id = ?`,
		a.Title, a.Content, a.CategoryID, a.UpdatedAt, a.ID)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("文章")
	}
	return nil
}

func (r *articleRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return apperr.Internal(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("文章")
	}
	return nil
}

func (r *articleRepo) List(ctx context.Context, f model.ArticleFilter) ([]model.Article, int64, error) {
	where, args := buildArticleWhere(f)

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*)`+articleFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}

	q := `SELECT ` + articleCols + articleFrom + where +
		` ORDER BY a.created_at DESC, a.id DESC LIMIT ? OFFSET ?`
	args = append(args, f.PageSize, f.Offset())
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, 0, err
		}
		articles = append(articles, *a)
	}
	return articles, total, rows.Err()
}

// IncrViewCounts 在单个事务内批量累加浏览量，将 N 次写合并为 1 个事务。
func (r *articleRepo) IncrViewCounts(ctx context.Context, counts map[int64]int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return apperr.Internal(err)
	}
	defer func() { _ = tx.Rollback() }() // 已提交后 Rollback 为无害空操作

	stmt, err := tx.PrepareContext(ctx,
		`UPDATE articles SET view_count = view_count + ? WHERE id = ?`)
	if err != nil {
		return apperr.Internal(err)
	}
	defer stmt.Close()

	for id, n := range counts {
		if _, err := stmt.ExecContext(ctx, n, id); err != nil {
			return apperr.Internal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// listTags 查询文章关联的标签。
func (r *articleRepo) listTags(ctx context.Context, articleID int64) ([]model.Tag, error) {
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

// buildArticleWhere 将筛选条件拼装为 WHERE 子句与参数（参数化查询，防注入）。
func buildArticleWhere(f model.ArticleFilter) (string, []any) {
	var conds []string
	var args []any
	if f.CategoryID > 0 {
		conds = append(conds, "a.category_id = ?")
		args = append(args, f.CategoryID)
	}
	if f.TagID > 0 {
		conds = append(conds, "a.id IN (SELECT article_id FROM article_tags WHERE tag_id = ?)")
		args = append(args, f.TagID)
	}
	if f.Keyword != "" {
		kw := "%" + escapeLike(f.Keyword) + `%`
		conds = append(conds, `(a.title LIKE ? ESCAPE '\' OR a.content LIKE ? ESCAPE '\')`)
		args = append(args, kw, kw)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// escapeLike 转义 LIKE 通配符，防止用户输入干扰匹配语义。
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// scanArticle 从单行扫描一篇文章。
func scanArticle(row interface{ Scan(...any) error }) (*model.Article, error) {
	var a model.Article
	err := row.Scan(&a.ID, &a.Title, &a.Content, &a.AuthorID, &a.Author,
		&a.CategoryID, &a.Category, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("文章")
		}
		return nil, apperr.Internal(err)
	}
	return &a, nil
}
