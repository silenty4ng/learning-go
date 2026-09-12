package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"igoblog/internal/database"
	"igoblog/internal/middleware"
	"igoblog/internal/repository"
	"igoblog/internal/service"
)

// newTestServer 用内存级临时数据库装配一个真实可请求的 HTTP 服务。
// 该测试覆盖「注册 → 登录 → 发文 → 评论」的端到端链路。
func newTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := database.Open(filepath.Join(t.TempDir(), "smoke.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	userRepo := repository.NewUserRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// 冒烟测试用同步投递语义即可：ViewCounter 只需实现 Add
	viewCounter := &noopViewIncrer{}

	deps := Deps{
		User:     service.NewUserService(userRepo, settingsRepo, "test-secret", time.Hour),
		Article:  service.NewArticleService(articleRepo, categoryRepo, tagRepo, viewCounter),
		Category: service.NewCategoryService(categoryRepo),
		Tag:      service.NewTagService(tagRepo),
		Comment:  service.NewCommentService(commentRepo, articleRepo),
		Admin:    service.NewAdminService(userRepo, settingsRepo),
		UserRepo: userRepo,
	}
	// 测试中放大限流阈值，避免干扰流程验证
	return NewRouter(deps, "test-secret", middleware.NewRateLimiter(100000, 100000))
}

// noopViewIncrer 浏览量投递的空实现（handler 冒烟测试不关注落库）。
type noopViewIncrer struct{}

func (n *noopViewIncrer) Add(int64) {}

// ---------- 测试辅助 ----------

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any, token string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
	return w, resp
}

func wantStatus(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, code, w.Body.String())
	}
}

func tokenOf(t *testing.T, resp map[string]any) string {
	t.Helper()
	data, _ := resp["data"].(map[string]any)
	if data == nil {
		t.Fatalf("no data in response: %v", resp)
	}
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("no token in response: %v", resp)
	}
	return token
}

// ---------- 端到端冒烟测试 ----------

func TestSmokeFullFlow(t *testing.T) {
	r := newTestServer(t)

	// 1. 注册
	w, resp := doJSON(t, r, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"username": "alice", "password": "password123"}, "")
	wantStatus(t, w, http.StatusCreated)
	if _, ok := resp["data"]; !ok {
		t.Fatalf("register response missing data: %v", resp)
	}

	// 2. 登录取 token
	w, resp = doJSON(t, r, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"username": "alice", "password": "password123"}, "")
	wantStatus(t, w, http.StatusOK)
	token := tokenOf(t, resp)

	// 2.5 修改昵称并在 /users/me 中生效
	w, resp = doJSON(t, r, http.MethodPut, "/api/v1/users/me",
		map[string]string{"nickname": "小艾"}, token)
	wantStatus(t, w, http.StatusOK)
	if resp["data"].(map[string]any)["nickname"] != "小艾" {
		t.Fatalf("nickname not updated: %v", resp)
	}

	// 3. 未认证创建分类 -> 401
	w, _ = doJSON(t, r, http.MethodPost, "/api/v1/categories",
		map[string]string{"name": "Tech"}, "")
	wantStatus(t, w, http.StatusUnauthorized)

	// 4. 创建分类与标签
	w, resp = doJSON(t, r, http.MethodPost, "/api/v1/categories",
		map[string]string{"name": "Tech"}, token)
	wantStatus(t, w, http.StatusCreated)
	catID := int(resp["data"].(map[string]any)["id"].(float64))

	w, resp = doJSON(t, r, http.MethodPost, "/api/v1/tags",
		map[string]string{"name": "go"}, token)
	wantStatus(t, w, http.StatusCreated)
	tagID := int(resp["data"].(map[string]any)["id"].(float64))

	// 5. 发表文章
	articleBody := map[string]any{
		"title": "Hello Go", "content": "learning go",
		"category_id": catID, "tag_ids": []int{tagID},
	}
	w, resp = doJSON(t, r, http.MethodPost, "/api/v1/articles", articleBody, token)
	wantStatus(t, w, http.StatusCreated)
	article := resp["data"].(map[string]any)
	articleID := int(article["id"].(float64))
	// 作者展示昵称优先（第 2.5 步已将 alice 的昵称设为"小艾"）
	if article["author"] != "小艾" || article["category"] != "Tech" {
		t.Fatalf("joined fields wrong: %v", article)
	}

	// 6. 分页列表
	w, resp = doJSON(t, r, http.MethodGet,
		fmt.Sprintf("/api/v1/articles?page=1&page_size=10&tag_id=%d", tagID), nil, "")
	wantStatus(t, w, http.StatusOK)
	data := resp["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Fatalf("list total = %v, want 1", data["total"])
	}

	// 7. 文章详情
	w, resp = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", articleID), nil, "")
	wantStatus(t, w, http.StatusOK)

	// 8. 发表评论
	w, resp = doJSON(t, r, http.MethodPost,
		fmt.Sprintf("/api/v1/articles/%d/comments", articleID),
		map[string]string{"content": "nice post"}, token)
	wantStatus(t, w, http.StatusCreated)
	commentID := int(resp["data"].(map[string]any)["id"].(float64))

	// 9. 评论列表
	w, resp = doJSON(t, r, http.MethodGet,
		fmt.Sprintf("/api/v1/articles/%d/comments", articleID), nil, "")
	wantStatus(t, w, http.StatusOK)
	if resp["data"].(map[string]any)["total"].(float64) != 1 {
		t.Fatalf("comment total should be 1: %v", resp)
	}

	// 10. 作者删除评论
	w, _ = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/comments/%d", commentID), nil, token)
	wantStatus(t, w, http.StatusOK)

	// 11. 删除文章后详情应 404
	w, _ = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/articles/%d", articleID), nil, token)
	wantStatus(t, w, http.StatusOK)
	w, resp = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/articles/%d", articleID), nil, "")
	wantStatus(t, w, http.StatusNotFound)
	if resp["code"] != "NOT_FOUND" {
		t.Fatalf("error code = %v, want NOT_FOUND", resp["code"])
	}
}

func TestSmokeValidationAndNotFound(t *testing.T) {
	r := newTestServer(t)

	// 缺参数 -> 400
	w, resp := doJSON(t, r, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"username": "x"}, "")
	wantStatus(t, w, http.StatusBadRequest)
	if resp["code"] != "VALIDATION_ERROR" {
		t.Fatalf("code = %v", resp["code"])
	}

	// 不存在的资源 -> 404
	w, resp = doJSON(t, r, http.MethodGet, "/api/v1/articles/999", nil, "")
	wantStatus(t, w, http.StatusNotFound)
	if resp["code"] != "NOT_FOUND" {
		t.Fatalf("code = %v", resp["code"])
	}

	// 非法路径参数 -> 400
	w, _ = doJSON(t, r, http.MethodGet, "/api/v1/articles/abc", nil, "")
	wantStatus(t, w, http.StatusBadRequest)
}
