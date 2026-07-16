# cube_sandbox_image_server

`cube_sandbox_image_server` 是一个独立的 go-zero 项目，依赖边界由当前目录下的 `go.mod` 和 `go.sum` 定义。

## API

接口契约维护在 `cube_sandbox_image_server.api` 中：

- `POST /api/v1/sandbox/tools`：先通过镜像 Registry V2 API 读取默认启动命令，再调用腾讯云 AGS SDK 创建自定义沙箱工具；获得 `ToolId` 后立即返回，不在服务端持续轮询状态。
- `GET /api/v1/sandbox/tools/status`：根据 `toolId` 查询一次腾讯云沙箱工具的当前状态，供调用方按需轮询。
- `GET /api/v1/images/sync`：将 `srcHub`、`dstHub`、`group`、`image`、`tag` 安全编码为查询参数，请求内部镜像同步服务并原样映射业务响应。
- `GET /readiness`：Kubernetes Readiness 探针，服务正常运行时直接返回 HTTP 200 和空响应体，不访问腾讯云、Registry 或镜像同步服务。

### API Key 鉴权

所有 `/api/v1` 业务接口都必须携带 `X-API-Key` 请求头，Key 由运行配置中的 `Auth.APIKeys` 列表提供。列表中的任意一个 Key 都可以通过鉴权，因此可以先增加新 Key、切换调用方，再删除旧 Key，实现平滑轮换。

缺少或传入无效 Key 时，接口返回 HTTP 401：

```json
{"code":"UNAUTHORIZED","message":"缺少或无效的 API Key"}
```

`GET /readiness` 不需要 API Key，Kubernetes 探针可以继续直接访问。API Key 不应出现在日志、README 或提交到版本库的配置示例中。

镜像同步客户端仅对网络错误、HTTP 429 和 5xx 进行有限次数重试，并具有单次超时、总超时和响应体大小上限。创建工具不会盲目重试，避免未设置 `clientToken` 时重复创建。

### `/api/v1/sandbox/tools` 请求参数

请求结构和校验规则以 `cube_sandbox_image_server.api` 为准；所有服务端缺省值统一定义在 `internal/config/defaults.go`。API 生成类型不再通过结构标签另行注入默认值。

| 类型 | 参数 | 说明/默认值 |
|---|---|---|
| 必填 | `toolName` | Tool 名称，同一 AppId 下必须唯一 |
| 必填 | `customConfiguration.image` | 可拉取的运行镜像地址 |
| 可选 | `networkConfiguration.vpcConfig.subnetIds` | 默认 `["subnet-21t3qpcr"]` |
| 可选 | `description` | 默认 `custom sandbox tool` |
| 可选 | `defaultTimeout` | 默认 `5m`，最大 `24h` |
| 可选 | `clientToken` | 默认不传；重试同一次创建时应保持不变 |
| 可选 | `roleArn` | 默认 `qcs::cam::uin/100012162316:roleName/aime-ags` |
| 可选 | `persistent` | 默认 `false` |
| 可选 | `tags` | 默认不传 |
| 可选 | `networkConfiguration.networkMode` | 默认 `VPC`；可选 `PUBLIC` / `VPC` / `SANDBOX` |
| 可选 | `networkConfiguration.vpcConfig.securityGroupIds` | 默认 `["sg-nkwdhmgy"]` |
| 可选 | `storageMounts` | 默认不挂载卷；省略或传空数组时不向云端传递挂载配置 |
| 可选 | `storageMounts[].name` / `mountPath` / `readOnly` | 显式添加挂载项时，缺省值为 `aime-agent-harness` / `/mnt/cos` / `false` |
| 可选 | `storageMounts[].storageSource.cos` | 显式添加挂载项时，Endpoint 缺省为 `aime-agent-cos-1300730068.cos.ap-shanghai.myqcloud.com`，BucketName 缺省为 `aime-agent-cos-1300730068`，BucketPath 缺省为 `/` |
| 可选 | `customConfiguration.imageRegistryType` | 默认 `enterprise`；可选 `enterprise` / `personal` |
| 可选 | `customConfiguration.command` | 默认为空数组；创建时使用镜像 `Config.Entrypoint`，为空时回退到 `Config.Cmd` |
| 可选 | `customConfiguration.args` / `env` | 默认不传 |
| 可选 | `customConfiguration.ports` | 默认 `[{"name":"envd","protocol":"TCP","port":49983}]` |
| 可选 | `customConfiguration.resources` | CPU 默认 `0.5`，Memory 默认 `1Gi` |
| 可选 | `customConfiguration.probe` | 默认 `/health:49999`、`HTTP`、Ready `30000ms`、Timeout `1000ms`、Period `300ms`、Success `1`、Failure `100` |
| 可选 | `customConfiguration.dnsConfig` | Servers/Searches/Options 均为空时不传 |

## 接口请求示例

以下示例默认服务运行在 `http://127.0.0.1:43999`。镜像地址、Tool 名称、角色 ARN 和 VPC 配置需要根据实际环境调整。

### 创建沙箱工具：最小请求

只传入必填字段，其他字段使用上表中的默认值：

```bash
curl -sS -X POST 'http://127.0.0.1:43999/api/v1/sandbox/tools' \
  -H 'X-API-Key: replace-with-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "toolName": "custom-sandbox-httpserver",
    "customConfiguration": {
      "image": "ths-shanghai-tcr.tencentcloudcr.com/ths/aime-sandbox-light-server:1.0.0.46-20260617190922.liziyue.36c86c4e.46.cubesandbox-image"
    }
  }'
```

每次创建请求都会实时访问 Registry，不跨请求缓存 manifest 或 config，用于确认镜像可访问且具有有效启动配置。`command` 默认为空数组；未显式传入时使用镜像 `Config.Entrypoint`，为空时回退到 `Config.Cmd`，显式传入时使用请求值。

最小请求不需要显式传入 `storageMounts`，默认不挂载卷。显式传入空数组的效果相同；需要挂载 COS 时才传入具体挂载项。

### 创建沙箱工具：完整配置

```bash
curl -sS -X POST 'http://127.0.0.1:43999/api/v1/sandbox/tools' \
  -H 'X-API-Key: replace-with-api-key' \
  -H 'Content-Type: application/json' \
  -d '{
    "toolName": "custom-sandbox-full-demo",
    "description": "custom code interpreter sandbox",
    "defaultTimeout": "10m",
    "clientToken": "create-tool-demo-001",
    "roleArn": "qcs::cam::uin/100012162316:roleName/aime-ags",
    "persistent": false,
    "tags": [
      {"key": "env", "value": "demo"}
    ],
    "networkConfiguration": {
      "networkMode": "PUBLIC"
    },
    "storageMounts": [
      {
        "name": "aime-agent-harness",
        "storageSource": {
          "cos": {
            "endpoint": "aime-agent-cos-1300730068.cos.ap-shanghai.myqcloud.com",
            "bucketName": "aime-agent-cos-1300730068",
            "bucketPath": "/"
          }
        },
        "mountPath": "/mnt/cos",
        "readOnly": false
      }
    ],
    "customConfiguration": {
      "image": "ths-shanghai-tcr.tencentcloudcr.com/ths/cube-sandbox/sandbox-code:latest",
      "imageRegistryType": "enterprise",
      "command": [],
      "env": [
        {"name": "CODE_INTERPRETER_HOST", "value": "0.0.0.0"},
        {"name": "CODE_INTERPRETER_PORT", "value": "49999"},
        {"name": "CODE_INTERPRETER_WORKDIR", "value": "/workspace"}
      ],
      "ports": [
        {"name": "envd", "protocol": "TCP", "port": 49983}
      ],
      "resources": {
        "cpu": "0.5",
        "memory": "1Gi"
      },
      "probe": {
        "httpGet": {
          "path": "/health",
          "port": 49999,
          "scheme": "HTTP"
        },
        "readyTimeoutMs": 30000,
        "probeTimeoutMs": 1000,
        "probePeriodMs": 300,
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
  -H 'X-API-Key: replace-with-api-key' \
  --data-urlencode 'srcHub=hub-dev.hexin.cn:9544' \
  --data-urlencode 'dstHub=ths-shanghai-tcr.tencentcloudcr.com' \
  --data-urlencode 'group=ths' \
  --data-urlencode 'image=aime-sandbox-light-server' \
  --data-urlencode 'tag=1.0.0.46-20260617190922.liziyue.36c86c4e.46.cubesandbox-image'
```

`image` 参数必须与 `ImageSync.AllowedImageNames` 中的一个镜像名完全一致。匹配时不包含 `srcHub`、`group` 和 `tag`，并区分大小写。不在白名单中的请求返回 HTTP 403：

```json
{"code":"IMAGE_NOT_ALLOWED","message":"镜像名不在同步白名单中"}
```

### 查询工具状态

创建接口返回 `toolId` 后，可查询一次当前状态：

```bash
curl -sS --get 'http://127.0.0.1:43999/api/v1/sandbox/tools/status' \
  -H 'X-API-Key: replace-with-api-key' \
  --data-urlencode 'toolId=sdt-xxxxxxxx'
```

项目根目录的 `sync_image_and_create_tool.sh` 提供 `sync`、`tool`、`watch` 和 `help` 子命令。脚本顶部已经写入与本地私有 YAML 一致的 API Key，默认访问 `http://127.0.0.1:43999`，运行时无需另外传入鉴权参数。`sync` 的镜像输入格式为 `<IMAGE_GROUP>/<IMAGE_NAME>:<IMAGE_TAG>`，目标仓库由脚本顶部的 `DST_HUB` 补全。

```bash
./sync_image_and_create_tool.sh sync [目标镜像路径]
./sync_image_and_create_tool.sh tool [完整镜像地址] [工具名称]
./sync_image_and_create_tool.sh tool --config <完整请求JSON文件>
./sync_image_and_create_tool.sh watch [toolId] [超时秒]
./sync_image_and_create_tool.sh help
```

`TOOL_STATUS_POLL_INTERVAL` 和 `TOOL_STATUS_POLL_TIMEOUT` 环境变量仍可调整轮询间隔和总超时。脚本不依赖 `jq`；鉴权失败时 `watch` 会立即退出，不会继续重试。PaaS 或其他部署环境必须单独配置自己的 `Auth.APIKeys`，本地生成的 Key 不会自动同步到远端。

## 配置

服务统一从 YAML 读取配置，默认运行文件为 `etc/cubesandboximageserver-api.yaml`。仓库只提交不含真实凭证的 `etc/cubesandboximageserver-api.example.yaml`，首次运行前执行：

```bash
make config
```

该命令会从示例生成已被 `.gitignore` 忽略的私有运行配置；如果目标文件已存在则保持原文件不变。下表是当前示例配置中的默认运行值：

| 配置项 | 当前值 | 含义 |
|---|---|---|
| `Name` | `cube_sandbox_image_server-api` | go-zero 服务名 |
| `Host` | `0.0.0.0` | 监听所有网卡 |
| `Port` | `43999` | HTTP 服务端口 |
| `Timeout` | `600000` | go-zero HTTP 请求总超时，单位毫秒，即 10 分钟 |
| `Auth.APIKeys` | 必填，无默认值 | `X-API-Key` 允许列表，至少配置一个非空 Key |
| `TencentCloud.SecretID` | 必填，无默认值 | 腾讯云 API 访问凭证 |
| `TencentCloud.SecretKey` | 必填，无默认值 | 腾讯云 API 访问凭证 |
| `TencentCloud.Region` | `ap-shanghai` | AGS 地域 |
| `TencentCloud.Endpoint` | `ags.tencentcloudapi.com` | AGS API 地址 |
| `TencentCloud.RequestTimeout` | `60s` | 创建工具或查询工具状态时，单次腾讯云 SDK 调用的超时 |
| `ImageSync.Endpoint` | `http://172.20.208.115/sync/image` | 镜像同步服务地址 |
| `ImageSync.AllowedImageNames` | `aime-sandbox-light-server`<br>`aime-harness-sandbox-image` | `/images/sync` 允许同步的镜像名白名单，至少配置一个非空值 |
| `ImageSync.RequestTimeout` | `600s` | `/images/sync` 整个同步流程的总超时，包含所有尝试和重试等待 |
| `ImageSync.AttemptTimeout` | `300s` | 每次上游 HTTP 请求的超时，包含连接、等待响应和读取响应体 |
| `ImageSync.MaxResponseBytes` | `1048576` | 上游响应体上限，即 1 MiB |
| `ImageSync.MaxAttempts` | `3` | 最多尝试次数，包含首次请求，可配范围为 1–10 |
| `ImageSync.RetryInterval` | `500ms` | 首次重试等待时间，后续指数退避，最长 5 秒 |
| `Registry.Username` | 必填，无默认值 | Registry 认证用户名 |
| `Registry.Password` | 必填，无默认值 | Registry 认证密码 |
| `Registry.RequestTimeout` | `10s` | 从 Registry 解析镜像 Entrypoint/Cmd 的总超时 |

旧的私有运行配置必须增加 `Auth.APIKeys` 和 `ImageSync.AllowedImageNames` 后才能启动。仓库中的示例只包含占位 API Key；当前本地私有 YAML 已写入与 Shell 一致的随机 Key。真实 Key 同时存在于受 Git 跟踪的 Shell 中，如果不希望共享该 Key，提交前应重新生成并同步更新本地 YAML。

`TencentCloud.StatusPollInterval` 和 `TencentCloud.StatusPollTimeout` 是历史字段。如果旧的私有配置中仍然保留这两项，当前配置结构也不会读取，服务端不再自动轮询。工具状态由调用方通过 `GET /api/v1/sandbox/tools/status` 按需查询。

运行配置会先创建 `internal/config/defaults.go` 中的集中缺省配置，再用 YAML 中显式填写的值覆盖。例如代码缺省值为 `TencentCloud.RequestTimeout=10s`、`ImageSync.RequestTimeout=20s`、`ImageSync.AttemptTimeout=5s`、`Registry.RequestTimeout=10s`；当前 YAML 已显式设置上表中的运行值，因此运行时以 YAML 覆盖值为准。

旧的 `TENCENTCLOUD_*`、`IMAGE_SYNC_ENDPOINT`、`REGISTRY_*` 环境变量不再参与服务配置加载，也不会覆盖 YAML。需要使用其他运行配置时，通过 `-f` 显式指定：

```bash
go run . -f /path/to/cubesandboximageserver-api.yaml
```

敏感字段必须在部署时填入，不要在 README、镜像或新的代码提交中写入真实凭证。

Registry 不再配置 Host 白名单，默认允许访问任意域名的镜像仓库和 Bearer token 服务。所有 Registry 相关请求仍强制使用 HTTPS，镜像为多架构时固定选择 `linux/amd64`；跨主机 manifest/config blob 重定向仍由 Registry 客户端处理，并确保不会把 Registry 的 `Authorization` 请求头发送到重定向目标。

`POST /api/v1/sandbox/tools` 响应中，`accepted=true` 表示腾讯云控制面已接受创建请求，`status=CREATING` 仅表示创建请求已经提交。最终状态需要通过 `GET /api/v1/sandbox/tools/status` 查询；异步创建失败时，该接口返回 `status=FAILED` 和 `statusReason`。

Registry 解析失败时接口仍返回 HTTP 200，但响应为 `accepted=false`、`status=REJECTED`，`errorCode` 为 `IMAGE_COMMAND_RESOLVE_FAILED`；此时不会调用腾讯云创建或状态查询接口。

## Development

所有命令都应当在当前目录执行：

```bash
make api          # 使用项目锁定的 goctl 重新生成代码和 yapi.json
make config       # 从示例生成私有运行配置，不覆盖已有文件
make check        # 验证 API、格式化、整理依赖、vet 和测试
make run          # 确保运行配置存在后启动服务
```

goctl 版本作为 Go tool 依赖锁定在当前项目的 `go.mod` 中，无需全局安装。
