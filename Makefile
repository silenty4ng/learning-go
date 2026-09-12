.PHONY: run build test test-race vet tidy clean

# 运行服务（Windows PowerShell 亦可直接执行: go run ./cmd/server）
run:
	go run ./cmd/server

# 编译全部包
build:
	go build ./...

# 全量测试
test:
	go test ./... -count=1

# 竞态检测（验证并发组件正确性）
test-race:
	go test ./... -race -count=1

# 静态检查
vet:
	go vet ./...

# 整理依赖
tidy:
	go mod tidy

# 清理本地数据库文件
clean:
	go clean
	-del blog.db blog.db-shm blog.db-wal
