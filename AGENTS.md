# Repository Guidelines

## 项目结构与模块组织

- `cmd/plane/main.go` 是 CLI 入口，`internal/commands/` 定义命令树和资源处理器。
- `internal/api/` 负责认证、HTTP 请求、参数校验、错误脱敏和图片传输。
- `internal/output/` 负责表格/JSON 输出、分页信息和终端颜色。
- `internal/markdown/` 负责简单 Markdown 到 Plane HTML 的转换。
- `references/work-item-description.md` 维护 Plane 需求正文的 HTML、样式和 Markdown 编写规范。
- `Makefile` 负责本地构建、测试、五平台交叉编译和文档-only skill zip 打包。
- `SKILL.md` 和 `references/` 是 skill zip 的源文件。用户可见命令变更时，同步更新 `README.md` 和 `SKILL.md`。

## 构建、测试与开发命令

- `make build` 构建当前平台的 `dist/plane`。
- `make test` 运行 `gofmt` 检查、`go test ./...` 和 `go vet ./...`。
- `make release` 构建五个平台二进制、`my-plane-skill.zip` 和 `SHA256SUMS`。
- `go run ./cmd/plane --help` 或 `dist/plane <command> --help` 检查命令注册和参数帮助，不会访问 Plane。
- 设置 `PLANE_API_KEY` 和 `PLANE_WORKSPACE` 后，运行 `dist/plane projects list -f json` 等定向冒烟测试。涉及写入的命令应使用临时项目或工作项。

推送 `v*.*.*` 标签时，CI 会运行 `make release` 并上传五个平台二进制和 skill zip。

## 代码风格与命名约定

使用 Go 1.22+、`gofmt` 和易读的 Go 命名约定。命令处理器按资源和行为拆分；保持表格/JSON 两种输出、现有参数和颜色降级行为。默认使用标准库，避免引入运行时依赖。

## 测试规范

每次修改至少运行 `make test`，并在适用时分别验证受影响命令的普通输出和 JSON 输出。除正常流程外，也要检查预期的 API 错误和空结果。新增测试放在对应 Go 包中，文件名使用 `*_test.go`。

## 安全与配置

从环境变量读取 `PLANE_API_KEY`、`PLANE_WORKSPACE` 以及可选的 `PLANE_BASE_URL`。严禁提交 `.env`、凭据或包含敏感信息的 API 响应，也不要在错误信息或调试输出中打印令牌。

## 提交与 Pull Request 规范

现有历史倾向使用简短的祈使句式主题，有时采用 `fix:` 或 `fix(scope):` 格式，例如 `fix(images): handle missing extension`。提交应保持单一目的。PR 需要说明行为变化、列出验证命令、在适用时关联 Issue，并同步更新 CLI 文档。终端输出格式变化时，应附上修改前后的示例。
