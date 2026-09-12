package repository

import (
	"context"
	"path/filepath"
	"testing"

	"igoblog/internal/database"
)

func TestSettingsRoundTrip(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// 迁移时应已写入默认值
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if all["registration_enabled"] != "true" {
		t.Fatalf("default registration_enabled = %q, want true", all["registration_enabled"])
	}

	// 更新（upsert）
	if err := repo.Set(ctx, "registration_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := repo.Set(ctx, "custom_key", "custom_value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	all, err = repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if all["registration_enabled"] != "false" || all["custom_key"] != "custom_value" {
		t.Fatalf("settings not persisted: %v", all)
	}
}

func TestEnsureAdmin(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "admin.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	// 首次调用：创建管理员并返回凭据
	user, pass, created, err := database.EnsureAdmin(db)
	if err != nil {
		t.Fatalf("EnsureAdmin: %v", err)
	}
	if !created || user == "" || len(pass) < 8 {
		t.Fatalf("unexpected result: user=%q passLen=%d created=%v", user, len(pass), created)
	}

	// 再次调用：已有管理员，不重复创建
	_, _, created, err = database.EnsureAdmin(db)
	if err != nil {
		t.Fatalf("EnsureAdmin second: %v", err)
	}
	if created {
		t.Fatal("admin already exists, should not create again")
	}
}
