package repository

import "time"

// timeNow 返回截断到秒的 UTC 时间，保证与 SQLite DATETIME 存储精度一致。
func timeNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
