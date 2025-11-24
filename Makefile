.PHONY: build run clean deps test

# 构建项目
build:
	go build -o bff-proxy main.go

# 运行项目
run:
	go run main.go

# 下载依赖
deps:
	go mod download
	go mod tidy

# 清理构建文件
clean:
	rm -f bff-proxy
	rm -rf logs/*.log

# 运行测试
test:
	go test ./...

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
vet:
	go vet ./...

