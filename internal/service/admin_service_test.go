package service

import (
	"context"
	"errors"
	"testing"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
)

// newTestAdminService 构造带两名管理员与一名普通用户的测试环境。
// 用户：1=admin(owner) 2=admin2 3=alice(user)
func newTestAdminService(t *testing.T) (AdminService, *mockUserRepo) {
	t.Helper()
	repo := newMockUserRepo()
	mustRegister(t, repo, "owner", model.RoleAdmin)  // id=1
	mustRegister(t, repo, "admin2", model.RoleAdmin) // id=2
	mustRegister(t, repo, "alice", model.RoleUser)   // id=3
	svc := NewAdminService(repo, newMockSettingsRepo(nil))
	return svc, repo
}

func mustRegister(t *testing.T, repo *mockUserRepo, name, role string) *model.User {
	t.Helper()
	u := &model.User{Username: name, PasswordHash: "hash", Role: role}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seed user %s: %v", name, err)
	}
	return u
}

func TestUpdateUserRoleSuccess(t *testing.T) {
	svc, _ := newTestAdminService(t)
	u, err := svc.UpdateUserRole(context.Background(), 1, 3, model.RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateUserRole() error = %v", err)
	}
	if u.Role != model.RoleAdmin {
		t.Fatalf("role = %q, want admin", u.Role)
	}
}

func TestUpdateUserRoleInvalid(t *testing.T) {
	svc, _ := newTestAdminService(t)
	// 非法角色值
	if _, err := svc.UpdateUserRole(context.Background(), 1, 3, "root"); err == nil {
		t.Fatal("invalid role should fail")
	}
	// 不能自我降级
	_, err := svc.UpdateUserRole(context.Background(), 1, 1, model.RoleUser)
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Code != apperr.CodeValidation {
		t.Fatalf("self demotion should be rejected, got %v", err)
	}
}

func TestUpdateUserRoleLastAdmin(t *testing.T) {
	// 只有一名管理员时，不能将其降级（自我降级被拒兜底了唯一管理员场景）
	repo := newMockUserRepo()
	mustRegister(t, repo, "solo", model.RoleAdmin) // id=1
	svc := NewAdminService(repo, newMockSettingsRepo(nil))

	if _, err := svc.UpdateUserRole(context.Background(), 1, 1, model.RoleUser); err == nil {
		t.Fatal("self demotion should fail")
	}
}

func TestDeleteUserSuccess(t *testing.T) {
	svc, repo := newTestAdminService(t)
	if err := svc.DeleteUser(context.Background(), 1, 3); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if _, err := repo.GetByID(context.Background(), 3); err == nil {
		t.Fatal("user should be gone")
	}
}

func TestDeleteUserGuards(t *testing.T) {
	svc, _ := newTestAdminService(t)

	// 不能删除自己
	err := svc.DeleteUser(context.Background(), 1, 1)
	var ae *apperr.AppError
	if !errors.As(err, &ae) || ae.Code != apperr.CodeValidation {
		t.Fatalf("self delete should be rejected, got %v", err)
	}

	// 删除最后一名管理员被拒：先降级 admin2，再删除 owner（自身）会被拦截，
	// 但删除另一个 admin 当只剩两名管理员时是允许的——先删 admin2 成功，
	// 之后 owner 成为唯一管理员，尝试删除 owner 时因"删除自己"被拒。
	if err := svc.DeleteUser(context.Background(), 1, 2); err != nil {
		t.Fatalf("delete second admin should succeed, got %v", err)
	}
	if err := svc.DeleteUser(context.Background(), 1, 1); err == nil {
		t.Fatal("deleting self (last admin) should fail")
	}
}

func TestSiteSettingsRoundTrip(t *testing.T) {
	svc, _ := newTestAdminService(t)
	ctx := context.Background()

	// 默认值：注册开放
	s, err := svc.GetSiteSettings(ctx)
	if err != nil {
		t.Fatalf("GetSiteSettings() error = %v", err)
	}
	if !s.RegistrationEnabled {
		t.Fatal("registration should default to enabled")
	}

	// 更新后读回
	updated, err := svc.UpdateSiteSettings(ctx, model.SiteSettings{
		SiteTitle: "我的博客", SiteDescription: "desc", RegistrationEnabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateSiteSettings() error = %v", err)
	}
	if updated.SiteTitle != "我的博客" || updated.RegistrationEnabled {
		t.Fatalf("unexpected updated settings: %+v", updated)
	}
	s, _ = svc.GetSiteSettings(ctx)
	if s.SiteTitle != "我的博客" || s.RegistrationEnabled {
		t.Fatalf("settings not persisted: %+v", s)
	}

	// 校验：空标题拒绝
	if _, err := svc.UpdateSiteSettings(ctx, model.SiteSettings{SiteTitle: " "}); err == nil {
		t.Fatal("empty site title should fail")
	}
}
