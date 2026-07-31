# cube_sandbox_image_server 开发规则

## 项目边界

- 将此目录视为一个独立的 Go 项目。
- 执行 Go 和 goctl 命令时，以此目录作为工作目录。
- 使用此目录中的 `go.mod` 和 `go.sum`，不要依赖父项目的模块依赖。
- 除非用户明确要求，否则不要修改父项目中的文件。

## 代码生成

- API 的唯一事实来源是 `cube_sandbox_image_server.api`。
- 使用以下命令重新生成 Go 代码：

  ```bash
  go tool goctl api go -api cube_sandbox_image_server.api -dir .
  ```

- 使用此项目 `go.mod` 中固定的 goctl 版本，不要依赖全局安装的版本。

- 如果 goctl 为已删除的路由遗留了生成的 handler 或 logic 文件，请删除这些过时文件。

## 代码质量

- handler 仅负责解析请求和写入响应；业务逻辑应放在 `internal/logic` 中。
- 将可复用的客户端和长生命周期依赖放在 `internal/svc.ServiceContext` 中，将配置放在 `internal/config` 中。
- 优先使用职责单一、命名清晰且符合 Go 惯例的函数，避免重复编写请求映射代码。
- 处理并包装错误时应提供有用的上下文。不要静默忽略错误，也不要记录凭证、令牌或其他敏感信息。
- 保持 `cube_sandbox_image_server.api` 中定义的 API 契约不变，并为非简单逻辑添加针对性测试。
- 对每个修改过的 Go 文件运行 `gofmt`。

## 验证

修改依赖或代码后，在此目录中执行：

```bash
go mod tidy
go vet ./...
go test ./...
```
