# 使用示例

## 基本使用

### 1. 启动服务

```bash
# 下载依赖
go mod download

# 运行服务
go run main.go
```

或者使用 Makefile：

```bash
make deps  # 下载依赖
make run   # 运行服务
```

### 2. 访问管理界面

打开浏览器访问：`http://localhost:8080/admin`

## 配置示例

### 示例 1：简单的 API 代理

将所有 `/api` 开头的请求转发到 `http://localhost:3000`：

```yaml
proxy:
  rules:
    - name: "API代理"
      match:
        path: "/api"
      target: "http://localhost:3000"
      timeout: 30
```

### 示例 2：根据环境转发

根据 Header 中的 `X-Environment` 转发到不同的服务器：

```yaml
proxy:
  rules:
    - name: "生产环境"
      match:
        path: "/api"
        headers:
          X-Environment: "production"
      target: "http://prod-server:3000"
    
    - name: "测试环境"
      match:
        path: "/api"
        headers:
          X-Environment: "test"
      target: "http://test-server:3000"
```

### 示例 3：根据 Query 参数转发

根据 `env` 查询参数转发：

```yaml
proxy:
  rules:
    - name: "开发环境"
      match:
        path: "/api"
        query:
          env: "dev"
      target: "http://localhost:3000"
    
    - name: "生产环境"
      match:
        path: "/api"
        query:
          env: "prod"
      target: "http://prod-server:3000"
```

### 示例 4：路径重写

将 `/api/v1/users` 重写为 `/users` 后转发：

```yaml
proxy:
  rules:
    - name: "用户API"
      match:
        path: "/api/v1/users"
      target: "http://localhost:3000"
      rewrite_path: "/users"
```

### 示例 5：POST 请求 Body 匹配

根据请求体中的 `type` 字段转发：

```yaml
proxy:
  rules:
    - name: "支付请求"
      match:
        path: "/api"
        method: "POST"
        body:
          type: "payment"
      target: "http://payment-server:3000"
    
    - name: "订单请求"
      match:
        path: "/api"
        method: "POST"
        body:
          type: "order"
      target: "http://order-server:3000"
```

## 测试代理

### 使用 curl 测试

```bash
# GET 请求
curl http://localhost:8080/api/users

# POST 请求
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"name": "test"}'

# 带 Header 的请求
curl http://localhost:8080/api/users \
  -H "X-Environment: test"

# 带 Query 参数的请求
curl "http://localhost:8080/api/users?env=dev"
```

### 测试 SSE

```bash
curl -N -H "Accept: text/event-stream" http://localhost:8080/api/events
```

## 查看日志

1. 通过 Web 界面查看：访问 `http://localhost:8080/admin`，切换到"日志查看"标签
2. 直接查看日志文件：`logs/bff-proxy.log`

日志文件是 JSON Lines 格式，每行一条日志记录。

## 常见问题

### 1. 端口被占用

修改 `config.yaml` 中的端口配置：

```yaml
server:
  port: 8081  # 改为其他端口
```

### 2. 配置不生效

- 确保配置文件格式正确（YAML 格式）
- 检查配置文件路径是否正确
- 查看控制台是否有错误信息

### 3. 请求没有匹配到规则

- 检查路径、方法、header、query、body 是否匹配
- 规则按顺序匹配，第一个匹配的规则会被使用
- 确保目标服务器地址正确

### 4. SSE 不工作

- 确保目标服务器支持 SSE
- 检查请求头中是否包含 `Accept: text/event-stream`
- 检查响应头中是否包含 `Content-Type: text/event-stream`

