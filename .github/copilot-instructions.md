本文件用于指导 AI Agent 在本仓库进行开发时的工程习惯与约束。

## 核心架构与技术栈

- **语言**: Go 1.25+
- **框架**: Gin (Web), GORM (ORM), Godotenv (Config)
- **测试**: Testify (Assertions), Glebarez SQLite (In-memory DB tests)
- **分层架构**:
  - `controllers/`: HTTP Handler，处理请求/响应，依赖 Service。
  - `services/`: 业务逻辑，依赖 **Repository 接口** (定义于 `services/repo_interfaces.go`)。
  - `repositories/`: 数据访问，实现接口，依赖 `*gorm.DB`。
  - `models/`: GORM 实体定义。
- **依赖注入**: 在 `main.go` 中手动组装 Controller -> Service -> Repository。

## 关键开发规范

### 1. 接口与可测试性
- **Service 层依赖接口**: 必须依赖 `services/repo_interfaces.go` 中的接口，禁止依赖具体 Repo 结构体。
- **Mock/Stub**: 单元测试中使用 Stub 实现接口（参考 `services/post_services_test.go`），避免 Mock 框架过度复杂化。
- **外部依赖注入**:
  - GeoIP: 使用 `utils.SetGeoIPHTTPClientForTest` 注入 Mock Client。
  - Time: 涉及时间敏感逻辑时，考虑注入 `Now()` 函数或传递时间参数。

### 2. 测试策略
- **命令**: `go test ./...` (禁止提交 `coverage.out` 等产物)。
- **数据库测试**:
  - 使用 `github.com/glebarez/sqlite` 纯 Go 驱动，避免 CGO。
  - 每个测试用例应使用独立的 DB 实例或事务回滚，避免并发冲突。
- **文件命名**: `*_test.go` 为主测试，`*_more_test.go` 用于边界/覆盖率补充测试。

### 3. 文档维护
- **API 文档**: 修改路由/参数时，**必须**同步更新 `doc/api_v1.md` 和 `test.api.http`。
- **部署文档**: 环境变量变更时，更新 `DEPLOYMENT.md`。
- **示例规范**: URL 中空格必须转义为 `%20`，JSON Payload 需符合 Gin Bind 规范。

## 变更原则

- **首先写文档**: 任何新功能或变更，先更新相关文档，再实现代码。需要用户认可文档变更后，再进行代码实现。
- **根因修复**: 拒绝仅为了过测试的 Hack。
- **API 稳定性**: 修改 Public API 需同步更新所有调用方。
- **永远测试**: 所有新功能和修复必须在完成后进行测试覆盖，并运行全工程测试。测试内容包括：修改部分代码的测试，全工程编译测试，全工程测试，以及覆盖率检查。要求新增加的功能测试覆盖率不低于 90%。

## 其他要求

- **禁止删除文件**: 只能要求用户对文件进行删除，禁止 call tools/使用终端 删除文件。