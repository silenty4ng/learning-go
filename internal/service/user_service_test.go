package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/pkg/jwtutil"
)

// mockUserRepo 以内存 map 模拟用户存储，用于脱离数据库测试业务逻辑。
// 这正是「service 只依赖 repository 接口」带来的可测试性。
type mockUserRepo struct {
	users  map[string]*model.User
	nextID int64
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: map[string]*model.User{}, nextID: 1}
}

func (m *mockUserRepo) Create(_ context.Context, u *model.User) error {
	if _, exists := m.users[u.Username]; exists {
		return apperr.Conflict("用户名 %q 已被占用", u.Username)
	}
	if u.Role == "" {
		u.Role = model.RoleUser
	}
	u.ID = m.nextID
	m.nextID++
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id int64) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, apperr.NotFound("用户")
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, apperr.NotFound("用户")
}

func (m *mockUserRepo) List(_ context.Context, page, pageSize int) ([]model.User, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (m *mockUserRepo) UpdateRole(_ context.Context, id int64, role string) error {
	u, err := m.GetByID(nil, id)
	if err != nil {
		return err
	}
	u.Role = role
	return nil
}

func (m *mockUserRepo) UpdateNickname(_ context.Context, id int64, nickname string) error {
	u, err := m.GetByID(nil, id)
	if err != nil {
		return err
	}
	u.Nickname = nickname
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id int64) error {
	for name, u := range m.users {
		if u.ID == id {
			delete(m.users, name)
			return nil
		}
	}
	return apperr.NotFound("用户")
}

func (m *mockUserRepo) CountAdmins(_ context.Context) (int, error) {
	n := 0
	for _, u := range m.users {
		if u.Role == model.RoleAdmin {
			n++
		}
	}
	return n, nil
}

// mockSettingsRepo 内存键值对，模拟站点设置。
type mockSettingsRepo struct{ kv map[string]string }

func newMockSettingsRepo(kv map[string]string) *mockSettingsRepo {
	if kv == nil {
		kv = map[string]string{}
	}
	return &mockSettingsRepo{kv: kv}
}

func (m *mockSettingsRepo) GetAll(_ context.Context) (map[string]string, error) {
	cp := make(map[string]string, len(m.kv))
	for k, v := range m.kv {
		cp[k] = v
	}
	return cp, nil
}

func (m *mockSettingsRepo) Set(_ context.Context, key, value string) error {
	m.kv[key] = value
	return nil
}

const testSecret = "test-secret"

func newTestUserService() UserService {
	return NewUserService(newMockUserRepo(), newMockSettingsRepo(nil), testSecret, time.Hour)
}

func TestRegisterSuccess(t *testing.T) {
	svc := newTestUserService()
	u, err := svc.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if u.ID == 0 || u.Username != "alice" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if u.Role != model.RoleUser {
		t.Fatalf("new user role = %q, want user", u.Role)
	}
	// 密码必须以哈希形式存储，而非明文
	if u.PasswordHash == "password123" || len(u.PasswordHash) < 40 {
		t.Fatalf("password not hashed: %q", u.PasswordHash)
	}
}

func TestRegisterClosed(t *testing.T) {
	// 注册开关关闭：注册应返回 403
	svc := NewUserService(
		newMockUserRepo(),
		newMockSettingsRepo(map[string]string{model.SettingRegistrationEnabled: "false"}),
		testSecret, time.Hour,
	)
	_, err := svc.Register(context.Background(), "alice", "password123")
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Status != 403 {
		t.Fatalf("want 403 when registration disabled, got %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, "bob", "password123"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	_, err := svc.Register(ctx, "bob", "password456")
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Code != apperr.CodeConflict {
		t.Fatalf("want Conflict error, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()
	cases := []struct{ name, username, password string }{
		{"username too short", "ab", "password123"},
		{"username invalid chars", "bad name!", "password123"},
		{"password too short", "alice", "123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Register(ctx, tc.username, tc.password); err == nil {
				t.Fatalf("expected validation error for %+v", tc)
			}
		})
	}
}

func TestLoginSuccess(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, "carol", "password123"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	token, err := svc.Login(ctx, "carol", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	claims, err := jwtutil.Parse(testSecret, token)
	if err != nil {
		t.Fatalf("Parse(token) error = %v", err)
	}
	if claims.Username != "carol" {
		t.Fatalf("claims.Username = %q, want carol", claims.Username)
	}
	if claims.Role != model.RoleUser {
		t.Fatalf("claims.Role = %q, want user", claims.Role)
	}
}

func TestUpdateProfile(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()
	u, err := svc.Register(ctx, "erin", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// 设置昵称
	updated, err := svc.UpdateProfile(ctx, u.ID, "小艾")
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.Nickname != "小艾" {
		t.Fatalf("nickname = %q, want 小艾", updated.Nickname)
	}

	// 清除昵称（空字符串回退用户名展示）
	cleared, err := svc.UpdateProfile(ctx, u.ID, "  ")
	if err != nil {
		t.Fatalf("UpdateProfile(clear) error = %v", err)
	}
	if cleared.Nickname != "" {
		t.Fatalf("nickname = %q, want empty", cleared.Nickname)
	}

	// 超长昵称拒绝
	if _, err := svc.UpdateProfile(ctx, u.ID, strings.Repeat("a", 31)); err == nil {
		t.Fatal("nickname longer than 30 should fail")
	}

	// 不存在的用户 -> 404
	_, err = svc.UpdateProfile(ctx, 999, "nick")
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Status != 404 {
		t.Fatalf("want 404 for missing user, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, "dave", "password123"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	_, err := svc.Login(ctx, "dave", "wrong-password")
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Status != 401 {
		t.Fatalf("want 401 Unauthorized, got %v", err)
	}
	// 用户不存在也应返回 401，避免暴露用户名是否存在
	_, err = svc.Login(ctx, "nobody", "password123")
	if !errors.As(err, &ae) || ae.Status != 401 {
		t.Fatalf("want 401 for unknown user, got %v", err)
	}
}
