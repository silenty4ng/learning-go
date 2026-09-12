// Package database 负责 SQLite 连接初始化与建表迁移。
package database

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"igoblog/internal/model"

	_ "modernc.org/sqlite" // 注册纯 Go 的 sqlite 驱动，无需 CGO
)

// Open 打开（必要时自动创建）SQLite 数据库，配置 PRAGMA 并执行迁移。
//
// 工程考量：
//   - SQLite 同一时刻只支持单写者，限制 MaxOpenConns=1 可从根源避免 SQLITE_BUSY；
//   - WAL 模式提升读写并发表现；foreign_keys 开启保证引用完整性；
//   - busy_timeout 缓解短暂锁竞争。
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// EnsureAdmin 保证站点至少存在一名管理员；首次建库时自动创建默认管理员，
// 返回其用户名与密码（仅创建时返回密码，调用方负责打印到控制台）。
// 密码来源：环境变量 BLOG_ADMIN_PASSWORD 优先，否则随机生成 12 位强密码。
func EnsureAdmin(db *sql.DB) (username, password string, created bool, err error) {
	var admins int
	if err = db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE role = ?`, model.RoleAdmin).Scan(&admins); err != nil {
		return "", "", false, fmt.Errorf("count admins: %w", err)
	}
	if admins > 0 {
		return "", "", false, nil
	}

	username = os.Getenv("BLOG_ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	password = os.Getenv("BLOG_ADMIN_PASSWORD")
	if password == "" {
		if password, err = randomPassword(12); err != nil {
			return "", "", false, fmt.Errorf("generate admin password: %w", err)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", false, fmt.Errorf("hash admin password: %w", err)
	}
	now := timeNow()
	if _, err = db.Exec(
		`INSERT INTO users (username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		username, string(hash), model.RoleAdmin, now, now); err != nil {
		return "", "", false, fmt.Errorf("insert admin: %w", err)
	}
	return username, password, true, nil
}

// timeNow 返回截断到秒的 UTC 时间，与 DATETIME 存储精度一致。
func timeNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

// randomPassword 生成指定长度的字母数字随机密码。
func randomPassword(n int) (string, error) {
	const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		out[i] = charset[idx.Int64()]
	}
	return string(out), nil
}

// migrate 执行幂等的建表迁移脚本。
func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	username      TEXT    NOT NULL UNIQUE,
	password_hash TEXT    NOT NULL,
	created_at    DATETIME NOT NULL,
	updated_at    DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
	id   INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS tags (
	id   INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS articles (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	title       TEXT    NOT NULL,
	content     TEXT    NOT NULL,
	author_id   INTEGER NOT NULL REFERENCES users(id),
	category_id INTEGER NOT NULL REFERENCES categories(id),
	view_count  INTEGER NOT NULL DEFAULT 0,
	created_at  DATETIME NOT NULL,
	updated_at  DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_articles_category ON articles(category_id);
CREATE INDEX IF NOT EXISTS idx_articles_created  ON articles(created_at);

CREATE TABLE IF NOT EXISTS article_tags (
	article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
	tag_id     INTEGER NOT NULL REFERENCES tags(id)     ON DELETE CASCADE,
	PRIMARY KEY (article_id, tag_id)
);
CREATE INDEX IF NOT EXISTS idx_article_tags_tag ON article_tags(tag_id);

CREATE TABLE IF NOT EXISTS comments (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	content    TEXT    NOT NULL,
	created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_comments_article ON comments(article_id, created_at);

CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	// 旧库升级：为 users 表补充 role / nickname 列（幂等）
	if err := ensureColumn(db, "users", "role", `role TEXT NOT NULL DEFAULT 'user'`); err != nil {
		return fmt.Errorf("ensure users.role: %w", err)
	}
	if err := ensureColumn(db, "users", "nickname", `nickname TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("ensure users.nickname: %w", err)
	}
	if err := seedSettings(db); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}
	return nil
}

// ensureColumn 检查列是否存在，缺失时执行 ALTER TABLE 补齐（支持旧库平滑升级）。
func ensureColumn(db *sql.DB, table, column, columnDDL string) error {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil // 已存在
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s`, table, columnDDL))
	return err
}

// seedSettings 写入站点设置的默认值（已存在时不覆盖）。
func seedSettings(db *sql.DB) error {
	_, err := db.Exec(`
INSERT OR IGNORE INTO settings (key, value) VALUES
	('site_title', 'iGoBlog'),
	('site_description', 'Go 语言实现的博客系统'),
	('registration_enabled', 'true')`)
	return err
}
