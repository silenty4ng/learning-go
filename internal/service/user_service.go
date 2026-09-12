package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"igoblog/internal/apperr"
	"igoblog/internal/model"
	"igoblog/internal/pkg/jwtutil"
	"igoblog/internal/repository"
)

// userService 是 UserService 的实现。
type userService struct {
	repo      repository.UserRepository
	settings  repository.SettingsRepository
	jwtSecret string
	jwtExpire time.Duration
}

// NewUserService 构造用户服务。
func NewUserService(repo repository.UserRepository, settings repository.SettingsRepository,
	jwtSecret string, expire time.Duration) UserService {
	return &userService{repo: repo, settings: settings, jwtSecret: jwtSecret, jwtExpire: expire}
}

// Register 注册新用户：密码经 bcrypt 加密后存储，明文密码不落库。
// 站点关闭注册时拒绝（403）。
func (s *userService) Register(ctx context.Context, username, password string) (*model.User, error) {
	open, err := s.registrationEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !open {
		return nil, apperr.Forbidden("本站已关闭新用户注册")
	}
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if len(password) < 6 || len(password) > 64 {
		return nil, apperr.BadRequest("密码长度须为 6~64 个字符")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	u := &model.User{Username: username, PasswordHash: string(hash), Role: model.RoleUser}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// registrationEnabled 读取注册开关设置；键缺失时默认允许注册。
func (s *userService) registrationEnabled(ctx context.Context) (bool, error) {
	all, err := s.settings.GetAll(ctx)
	if err != nil {
		return false, err
	}
	return all[model.SettingRegistrationEnabled] != "false", nil
}

// Login 校验用户名密码，成功后签发 JWT。
// 用户不存在与密码错误返回相同消息，避免暴露用户名是否存在。
func (s *userService) Login(ctx context.Context, username, password string) (string, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		var ae *apperr.AppError
		if errors.As(err, &ae) && ae.Code == apperr.CodeNotFound {
			return "", apperr.Unauthorized("用户名或密码错误")
		}
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", apperr.Unauthorized("用户名或密码错误")
	}
	token, err := jwtutil.Generate(s.jwtSecret, s.jwtExpire, u.ID, u.Username, u.Role)
	if err != nil {
		return "", apperr.Internal(err)
	}
	return token, nil
}

// Profile 查询用户完整资料。
func (s *userService) Profile(ctx context.Context, userID int64) (*model.User, error) {
	return s.repo.GetByID(ctx, userID)
}

// UpdateProfile 修改当前用户的昵称；空字符串表示清除昵称（展示回退到用户名）。
func (s *userService) UpdateProfile(ctx context.Context, userID int64, nickname string) (*model.User, error) {
	nickname = strings.TrimSpace(nickname)
	if len(nickname) > 30 {
		return nil, apperr.BadRequest("昵称不超过 30 字符")
	}
	if err := s.repo.UpdateNickname(ctx, userID, nickname); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID)
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return apperr.BadRequest("用户名长度须为 3~32 个字符")
	}
	for _, r := range username {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-'
		if !ok {
			return apperr.BadRequest("用户名仅允许字母、数字、下划线与连字符")
		}
	}
	return nil
}
