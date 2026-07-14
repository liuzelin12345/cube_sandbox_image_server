# cube_sandbox_image_server

`cube_sandbox_image_server` 是一个独立的 go-zero 项目，依赖边界由当前目录下的 `go.mod` 和 `go.sum` 定义。

## API

接口契约维护在 `cube_sandbox_image_server.api` 中：

- `POST /api/v1/sandbox/tools`：调用腾讯云 AGS SDK 创建自定义沙箱工具。接口在获得 `ToolId` 后会轮询状态，返回 `ACTIVE`、`FAILED` 或超时时的最后已知状态。
- `GET /api/v1/images/sync`：将 `srcHub`、`dstHub`、`group`、`image`、`tag` 安全编码为查询参数，请求内部镜像同步服务并原样映射业务响应。

镜像同步客户端仅对网络错误、HTTP 429 和 5xx 进行有限次数重试，并具有单次超时、总超时和响应体大小上限。创建工具不会盲目重试，避免未设置 `clientToken` 时重复创建。

### `/api/v1/sandbox/tools` 请求参数

以 `cmd/create-tool/.env.templete` 中的分类和默认值为准。

| 类型 | 参数 | 说明/默认值 |
|---|---|---|
| 必填 | `toolName` | Tool 名称，同一 AppId 下必须唯一 |
| 必填 | `customConfiguration.image` | 可拉取的运行镜像地址 |
| 必填 | `customConfiguration.command` | 非空启动命令数组 |
| 条件必填 | `networkConfiguration.vpcConfig.subnetIds` | `networkMode=VPC` 时至少一个 |
| 可选 | `description` | 默认不传 |
| 可选 | `defaultTimeout` | 默认 `5m`，最大 `24h` |
| 可选 | `clientToken` | 默认不传；重试同一次创建时应保持不变 |
| 可选 | `roleArn` | 默认 `qcs::cam::uin/100032159895:roleName/sandbox_test` |
| 可选 | `persistent` | 默认 `false` |
| 可选 | `tags` | 默认不传 |
| 可选 | `networkConfiguration.networkMode` | 默认 `PUBLIC`；可选 `PUBLIC` / `VPC` / `SANDBOX` |
| 可选 | `networkConfiguration.vpcConfig.securityGroupIds` | 默认不传 |
| 可选 | `customConfiguration.imageRegistryType` | 默认 `enterprise`；可选 `enterprise` / `personal` |
| 可选 | `customConfiguration.args` / `env` | 默认不传 |
| 可选 | `customConfiguration.ports` | 默认 `[{"name":"http","protocol":"TCP","port":49999}]` |
| 可选 | `customConfiguration.resources` | CPU 默认 `1`，Memory 默认 `2Gi` |
| 可选 | `customConfiguration.probe` | 默认 `/health:49999`、`HTTP`、Ready `30000ms`、Timeout/Period `1000ms`、Success `1`、Failure `100` |
| 可选 | `customConfiguration.dnsConfig` | Servers/Searches/Options 均为空时不传 |

## 接口请求示例

以下示例默认服务运行在 `http://127.0.0.1:43999`。镜像地址、Tool 名称、角色 ARN 和 VPC 配置需要根据实际环境调整。

### 创建沙箱工具：最小请求

只传入必填字段，其他字段使用上表中的默认值：

```bash
curl -sS -X POST 'http://127.0.0.1:43999/api/v1/sandbox/tools' \
  -H 'Content-Type: application/json' \
  -d '{
    "toolName": "custom-sandbox-httpserver",
    "customConfiguration": {
      "image": "ths-shanghai-tcr.tencentcloudcr.com/ths/aime-sandbox-light-server:1.0.0.46-20260617190922.liziyue.36c86c4e.46.cubesandbox-image",
      "command": ["/usr/local/bin/start-lightweight-code-interpreter.sh"]
    }
  }'
```

### 创建沙箱工具：完整配置

```bash
curl -sS -X POST 'http://127.0.0.1:43999/api/v1/sandbox/tools' \
  -H 'Content-Type: application/json' \
  -d '{
    "toolName": "custom-sandbox-full-demo",
    "description": "custom code interpreter sandbox",
    "defaultTimeout": "10m",
    "clientToken": "create-tool-demo-001",
    "roleArn": "qcs::cam::uin/100032159895:roleName/sandbox_test",
    "persistent": false,
    "tags": [
      {"key": "env", "value": "demo"}
    ],
    "networkConfiguration": {
      "networkMode": "PUBLIC"
    },
    "customConfiguration": {
      "image": "ths-shanghai-tcr.tencentcloudcr.com/ths/cube-sandbox/sandbox-code:latest",
      "imageRegistryType": "enterprise",
      "command": ["/usr/local/bin/start-lightweight-code-interpreter.sh"],
      "env": [
        {"name": "CODE_INTERPRETER_HOST", "value": "0.0.0.0"},
        {"name": "CODE_INTERPRETER_PORT", "value": "49999"},
        {"name": "CODE_INTERPRETER_WORKDIR", "value": "/workspace"}
      ],
      "ports": [
        {"name": "http", "protocol": "TCP", "port": 49999},
        {"name": "management", "protocol": "TCP", "port": 49983}
      ],
      "resources": {
        "cpu": "1",
        "memory": "2Gi"
      },
      "probe": {
        "httpGet": {
          "path": "/health",
          "port": 49999,
          "scheme": "HTTP"
        },
        "readyTimeoutMs": 30000,
        "probeTimeoutMs": 1000,
        "probePeriodMs": 1000,
        "successThreshold": 1,
        "failureThreshold": 100
      }
    }
  }'
```

`clientToken` 用于保证创建幂等：重试同一次创建时保持不变，发起新的创建请求时应换用新值。

VPC 模式下，`networkConfiguration` 需要改为：

```json
{
  "networkMode": "VPC",
  "vpcConfig": {
    "subnetIds": ["subnet-xxxxxxxx"],
    "securityGroupIds": ["sg-xxxxxxxx"]
  }
}
```

### 同步镜像

```bash
curl -sS --get 'http://127.0.0.1:43999/api/v1/images/sync' \
  --data-urlencode 'srcHub=hub-dev.hexin.cn:9544' \
  --data-urlencode 'dstHub=ths-shanghai-tcr.tencentcloudcr.com' \
  --data-urlencode 'group=ths' \
  --data-urlencode 'image=aime-sandbox-light-server' \
  --data-urlencode 'tag=1.0.0.46-20260617190922.liziyue.36c86c4e.46.cubesandbox-image'
```

## 配置

启动前必须通过进程环境变量提供腾讯云凭证：

```bash
export TENCENTCLOUD_SECRET_ID='...'
export TENCENTCLOUD_SECRET_KEY='...'
```

默认镜像同步地址为 `http://172.20.208.115/sync/image`，可以通过 `IMAGE_SYNC_ENDPOINT` 覆盖。超时、重试次数和响应体上限在 `etc/cubesandboximageserver-api.yaml` 中配置。服务不会自动读取 `.env`，如果本地使用 `.env`，需要先将其导入当前 shell。

`POST /api/v1/sandbox/tools` 响应中，`accepted=true` 表示腾讯云控制面已接受创建请求；异步创建仍可能最终返回 `status=FAILED`，此时可通过 `statusReason`、`errorCode` 和 `errorMessage` 查看原因。

## Development

所有命令都应当在当前目录执行：

```bash
make api          # 使用项目锁定的 goctl 重新生成代码
make check        # 验证 API、格式化、整理依赖、vet 和测试
make run          # 启动服务
```

goctl 版本作为 Go tool 依赖锁定在当前项目的 `go.mod` 中，无需全局安装。
