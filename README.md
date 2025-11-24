# BFF Proxy

一个功能强大的 BFF（Backend For Frontend）代理服务，用于开发测试环境。支持根据多种条件（header、path、query、body）将请求转发到不同的后端服务器，并提供 Web 界面进行配置管理和日志查看。

## 功能特性

- ✅ **HTTP 代理转发**：支持所有 HTTP 方法（GET、POST、PUT、DELETE 等）
- ✅ **SSE 支持**：支持 Server-Sent Events 流式传输
- ✅ **灵活的路由规则**：根据 path、method、header、query、body 参数匹配
- ✅ **配置热加载**：修改配置文件后自动重新加载，无需重启服务
- ✅ **Web 管理界面**：通过浏览器进行配置管理和日志查看
- ✅ **详细日志记录**：记录请求参数、响应结果、耗时、下游信息等
- ✅ **路径重写**：支持请求路径重写功能

## 快速开始

### 安装依赖

```bash
go mod download
```

### 配置文件

项目根目录下的 `config.yaml` 是配置文件，示例配置如下：

```yaml
server:
  port: 8080

proxy:
  rules:
    - name: "默认API代理"
      match:
        path: "/api"
        method: ""
      target: "http://localhost:3000"
      timeout: 30
      headers: {}
      rewrite_path: ""

log:
  level: "info"
  file: "logs/bff-proxy.log"
  max_size: 100
  max_backups: 10
  max_age: 30
```

### 运行服务

```bash
go run main.go
```

服务启动后，访问 `http://localhost:8080/admin` 进入管理界面。

## 配置说明

### 代理规则配置

每个代理规则包含以下字段：

- **name**: 规则名称（用于标识）
- **match**: 匹配条件
  - **path**: 路径匹配（支持前缀匹配，如 `/api`）
  - **method**: HTTP 方法（留空表示匹配所有方法）
  - **headers**: Header 匹配（键值对）
  - **query**: Query 参数匹配（键值对）
  - **body**: Body 参数匹配（仅支持 JSON 格式）
- **target**: 目标服务器地址
- **timeout**: 超时时间（秒，默认 30）
- **headers**: 额外添加的请求头
- **rewrite_path**: 路径重写（可选）

### 匹配规则示例

#### 1. 根据路径匹配

```yaml
match:
  path: "/api/users"
  method: "GET"
target: "http://localhost:3000"
```

#### 2. 根据 Header 匹配

```yaml
match:
  path: "/api"
  headers:
    X-API-Version: "v2"
target: "http://localhost:3001"
```

#### 3. 根据 Query 参数匹配

```yaml
match:
  path: "/api"
  query:
    env: "test"
target: "http://localhost:3002"
```

#### 4. 根据 Body 参数匹配

```yaml
match:
  path: "/api"
  method: "POST"
  body:
    type: "payment"
target: "http://localhost:3003"
```

## Web 管理界面

访问 `http://localhost:8080/admin` 可以：

1. **配置管理**
   - 查看当前配置
   - 添加、编辑、删除代理规则
   - 修改服务端口和日志配置
   - 保存配置（支持热加载）

2. **日志查看**
   - 查看最近的请求日志
   - 查看请求详情（请求头、请求体、响应体）
   - 查看耗时和状态码
   - 查看目标服务器信息

## API 接口

### 获取配置

```
GET /admin/api/config
```

### 更新配置

```
POST /admin/api/config
Content-Type: application/json

{
  "server": { "port": 8080 },
  "proxy": { "rules": [...] },
  "log": { ... }
}
```

### 获取日志

```
GET /admin/api/logs?limit=100
```

## 日志格式

日志以 JSON 格式记录在文件中，包含以下字段：

```json
{
  "start_time": "2025-01-01T12:00:00Z",
  "end_time": "2025-01-01T12:00:01Z",
  "duration": 1000000000,
  "method": "GET",
  "path": "/api/users",
  "query": "id=123",
  "headers": { "Content-Type": "application/json" },
  "body": "{}",
  "status_code": 200,
  "response_body": "{\"data\": [...]}",
  "target": "http://localhost:3000",
  "rule_name": "默认API代理"
}
```

## 项目结构

```
BFF-proxy/
├── main.go                 # 入口文件
├── config.yaml            # 配置文件
├── go.mod                 # Go 模块文件
├── internal/
│   ├── config/           # 配置管理
│   │   └── config.go
│   ├── proxy/            # 代理转发
│   │   └── proxy.go
│   ├── logger/           # 日志记录
│   │   └── logger.go
│   └── web/              # Web UI
│       └── web.go
└── web/
    └── static/           # 静态文件
        └── index.html
```

## 开发

### 添加新功能

1. 修改 `internal/` 目录下的相应模块
2. 更新配置文件结构（如需要）
3. 更新 Web UI（如需要）

### 构建

```bash
go build -o bff-proxy main.go
```

## 注意事项

1. 配置文件修改后会自动热加载，无需重启服务
2. 日志文件会自动轮转，根据配置保留指定天数的日志
3. SSE 请求会进行流式传输，响应体在日志中显示为 `[SSE Stream]`
4. Body 匹配仅支持 JSON 格式的请求体

## License

MIT License
