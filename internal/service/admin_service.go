package service

import (
	"context"
	"strconv"
	"strings"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/repository"
)

// adminService 是 AdminService 的实现：用户管理与站点设置。
type adminService struct {
	users    repository.UserRepository
	settings repository.SettingsRepository
}

// NewAdminService 构造管理员服务。
func NewAdminService(users repository.UserRepository, settings repository.SettingsRepository) AdminService {
	return &adminService{users: users, settings: settings}
}

func (s *adminService) ListUsers(ctx context.Context, page, pageSize int) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.users.List(ctx, page, pageSize)
}

// UpdateUserRole 修改用户角色。
// 安全约束：不能自我降级（避免误操作失去管理权）；站点至少保留一名管理员。
func (s *adminService) UpdateUserRole(ctx context.Context, operatorID, targetID int64, role string) (*model.User, error) {
	if role != model.RoleUser && role != model.RoleAdmin {
		return nil, apperr.BadRequest("角色仅允许 %s 或 %s", model.RoleUser, model.RoleAdmin)
	}
	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if targetID == operatorID && role != model.RoleAdmin {
		return nil, apperr.BadRequest("不能将自己降级为普通用户")
	}
	if target.IsAdmin() && role == model.RoleUser {
		n, err := s.users.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, apperr.Conflict("站点至少需要保留一名管理员")
		}
	}
	if err := s.users.UpdateRole(ctx, targetID, role); err != nil {
		return nil, err
	}
	return s.users.GetByID(ctx, targetID)
}

// DeleteUser 删除用户及其全部文章与评论。
// 安全约束：不能删除自己；不能删除最后一名管理员。
func (s *adminService) DeleteUser(ctx context.Context, operatorID, targetID int64) error {
	if targetID == operatorID {
		return apperr.BadRequest("不能删除自己的账号")
	}
	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return err
	}
	if target.IsAdmin() {
		n, err := s.users.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return apperr.Conflict("站点至少需要保留一名管理员，请先创建其他管理员")
		}
	}
	return s.users.Delete(ctx, targetID)
}

func (s *adminService) GetSiteSettings(ctx context.Context) (model.SiteSettings, error) {
	all, err := s.settings.GetAll(ctx)
	if err != nil {
		return model.SiteSettings{}, err
	}
	return parseSettings(all), nil
}

func (s *adminService) UpdateSiteSettings(ctx context.Context, in model.SiteSettings) (model.SiteSettings, error) {
	in.SiteTitle = strings.TrimSpace(in.SiteTitle)
	in.SiteDescription = strings.TrimSpace(in.SiteDescription)
	if in.SiteTitle == "" || len(in.SiteTitle) > 50 {
		return model.SiteSettings{}, apperr.BadRequest("站点名称不能为空且不超过 50 字符")
	}
	if len(in.SiteDescription) > 200 {
		return model.SiteSettings{}, apperr.BadRequest("站点描述不超过 200 字符")
	}

	updates := map[string]string{
		model.SettingSiteTitle:           in.SiteTitle,
		model.SettingSiteDescription:     in.SiteDescription,
		model.SettingRegistrationEnabled: strconv.FormatBool(in.RegistrationEnabled),
	}
	for k, v := range updates {
		if err := s.settings.Set(ctx, k, v); err != nil {
			return model.SiteSettings{}, err
		}
	}
	return in, nil
}

// parseSettings 将键值对转换为强类型设置，键缺失时给出默认值。
func parseSettings(all map[string]string) model.SiteSettings {
	return model.SiteSettings{
		SiteTitle:           orDefault(all[model.SettingSiteTitle], "iGoBlog"),
		SiteDescription:     orDefault(all[model.SettingSiteDescription], ""),
		RegistrationEnabled: all[model.SettingRegistrationEnabled] != "false",
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
