#!/usr/bin/env bash

set -Eeuo pipefail

# 用法：
#   ./sync_image_and_create_tool.sh sync [目标镜像路径]       格式：<IMAGE_GROUP>/<IMAGE_NAME>:<IMAGE_TAG>
#   ./sync_image_and_create_tool.sh tool [镜像地址] [工具名称]
#   ./sync_image_and_create_tool.sh tool --config <完整请求JSON文件>
#   ./sync_image_and_create_tool.sh watch [toolId] [超时秒]
#   ./sync_image_and_create_tool.sh help
#
# tool 完整接口参数示例（保存为 JSON 文件后通过 --config 传入，不传的参数项采用下面的默认值）：
# {
#   "toolName": "custom-sandbox-httpserver",          //必填
#   "description": "custom sandbox tool",
#   "defaultTimeout": "5m",
#   "clientToken": "",
#   "roleArn": "qcs::cam::uin/100012162316:roleName/aime-ags",
#   "persistent": false,
#   "tags": [],
#   "networkConfiguration": {
#     "networkMode": "VPC",
#     "vpcConfig": {
#       "subnetIds": [
#         "subnet-21t3qpcr"
#       ],
#       "securityGroupIds": [
#         "sg-nkwdhmgy"
#       ]
#     }
#   },
#   "storageMounts": [],
#   "customConfiguration": {
#     "image": "registry.example.com/group/image:tag",        //必填
#     "imageRegistryType": "enterprise",
#     "command": [],
#     "args": [],
#     "env": [],
#     "ports": [
#       {
#         "name": "envd",
#         "protocol": "TCP",
#         "port": 49983
#       }
#     ],
#     "resources": {
#       "cpu": "0.5",
#       "memory": "1Gi"
#     },
#     "probe": {
#       "httpGet": {
#         "path": "/health",
#         "port": 49999,
#         "scheme": "HTTP"
#       },
#       "readyTimeoutMs": 30000,
#       "probeTimeoutMs": 1000,
#       "probePeriodMs": 300,
#       "successThreshold": 1,
#       "failureThreshold": 100
#     },
#     "dnsConfig": {}
#   }
# }

# ─── 参数区：均可通过同名环境变量覆盖 ────────────────────────────────────────

# 必填：sync 使用分组/镜像:标签，tool 使用完整镜像地址；为空时交互输入。
IMAGE_REFERENCE="${IMAGE_REFERENCE:-}"
# 必填：tool 创建的工具名称；为空时交互输入。
TOOL_NAME="${TOOL_NAME:-}"
# 必填：watch 查询的工具 ID；为空时交互输入。
TOOL_ID="${TOOL_ID:-}"
# 可选：完整创建工具请求的 JSON 文件；设置后不再使用 IMAGE_REFERENCE 和 TOOL_NAME。
TOOL_CONFIG_FILE="${TOOL_CONFIG_FILE:-}"
# 必填：必须与本地服务 Auth.APIKeys 中的一个值一致。
API_KEY="apk_36a529b062e7d9ab8843f7b902565920843b526500e0824bd498f1f200c5dfc9"

# API 服务基地址，不包含末尾的 /。
API_BASE_URL="${API_BASE_URL:-http://paas.myhexin.com/sandbox-adapter}"
# 同步镜像时使用的源镜像仓库。
SRC_HUB="${SRC_HUB:-hub-dev.hexin.cn:9544}"
# 同步镜像时使用的目标镜像仓库，用户无需在镜像路径中填写。
DST_HUB="${DST_HUB:-aime-agent-tcr.tencentcloudcr.com}"
# 镜像同步接口路径。
SYNC_API_PATH="${SYNC_API_PATH:-/api/v1/images/sync}"
# 沙箱工具创建接口路径。
TOOL_API_PATH="${TOOL_API_PATH:-/api/v1/sandbox/tools}"
# 沙箱工具状态查询接口路径。
TOOL_STATUS_API_PATH="${TOOL_STATUS_API_PATH:-/api/v1/sandbox/tools/status}"
# watch 查询间隔，单位为秒。
TOOL_STATUS_POLL_INTERVAL="${TOOL_STATUS_POLL_INTERVAL:-5}"
# tool 自动 watch 及 watch 命令的默认超时时间，单位为秒。
TOOL_STATUS_POLL_TIMEOUT="${TOOL_STATUS_POLL_TIMEOUT:-300}"

API_BASE_URL="${API_BASE_URL%/}"

require_command() {
  local command_name="$1"

  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "错误：未找到依赖命令 ${command_name}" >&2
    return 1
  fi
}

validate_api_key() {
  if [[ -z "${API_KEY//[[:space:]]/}" || "${API_KEY}" == "__GENERATED_API_KEY__" ]]; then
    echo "错误：请先在脚本顶部配置 API_KEY" >&2
    return 1
  fi
}

validate_positive_integer() {
  local value="$1"
  local name="$2"

  if [[ ! "${value}" =~ ^[1-9][0-9]*$ ]]; then
    echo "错误：${name} 必须是正整数秒数" >&2
    return 1
  fi
}

read_required() {
  local prompt="$1"
  local variable_name="$2"
  local input_value

  while true; do
    if ! IFS= read -r -p "${prompt}: " input_value; then
      echo >&2
      echo "错误：未读取到输入" >&2
      return 1
    fi
    if [[ -n "${input_value//[[:space:]]/}" ]]; then
      printf -v "${variable_name}" '%s' "${input_value}"
      return 0
    fi
    echo "输入不能为空，请重新输入。" >&2
  done
}

resolve_required() {
  local current_value="$1"
  local prompt="$2"
  local variable_name="$3"

  if [[ -n "${current_value//[[:space:]]/}" ]]; then
    printf -v "${variable_name}" '%s' "${current_value}"
    return 0
  fi
  read_required "${prompt}" "${variable_name}"
}

parse_image_reference() {
  local image_reference="$1"
  local image_group_variable="$2"
  local image_name_variable="$3"
  local image_tag_variable="$4"
  local pattern='^([^/[:space:]]+)/([^/:[:space:]]+):([^/:[:space:]]+)$'

  if [[ ! "${image_reference}" =~ ${pattern} ]]; then
    return 1
  fi

  printf -v "${image_group_variable}" '%s' "${BASH_REMATCH[1]}"
  printf -v "${image_name_variable}" '%s' "${BASH_REMATCH[2]}"
  printf -v "${image_tag_variable}" '%s' "${BASH_REMATCH[3]}"
}

resolve_sync_image_reference() {
  local current_value="$1"
  local image_reference_variable="$2"
  local image_group_variable="$3"
  local image_name_variable="$4"
  local image_tag_variable="$5"
  local value="${current_value}"

  while true; do
    if [[ -z "${value//[[:space:]]/}" ]]; then
      printf '%s\n' \
        "请输入目标镜像路径" \
        "格式：<IMAGE_GROUP>/<IMAGE_NAME>:<IMAGE_TAG>"
      read_required "目标镜像路径" value || return 1
    fi

    if parse_image_reference \
      "${value}" \
      "${image_group_variable}" \
      "${image_name_variable}" \
      "${image_tag_variable}"; then
      printf -v "${image_reference_variable}" '%s' "${value}"
      return 0
    fi

    echo "错误：镜像路径格式必须为 <IMAGE_GROUP>/<IMAGE_NAME>:<IMAGE_TAG>" >&2
    value=""
  done
}

print_response() {
  local response="$1"

  printf '%s\n' "${response}"
}

extract_json_string() {
  local json="$1"
  local key="$2"
  local pattern

  pattern="\"${key}\"[[:space:]]*:[[:space:]]*\"([^\"]*)\""
  if [[ "${json}" =~ ${pattern} ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return 0
  fi
  return 1
}

json_escape() {
  local value="$1"

  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  value="${value//$'\t'/\\t}"
  printf '%s' "${value}"
}

# sync：将源仓库中的同名镜像同步到指定的目标镜像地址。
cmd_sync() {
  local image_reference="${1:-${IMAGE_REFERENCE}}"
  local IMAGE_GROUP IMAGE_NAME IMAGE_TAG
  local sync_response sync_status synced_image
  local sync_code_pattern='"code"[[:space:]]*:[[:space:]]*0[[:space:]]*[,}]'

  if [[ $# -gt 1 ]]; then
    echo "错误：用法：$0 sync [目标镜像地址]" >&2
    return 1
  fi

  require_command curl || return 1
  validate_api_key || return 1
  resolve_sync_image_reference \
    "${image_reference}" image_reference IMAGE_GROUP IMAGE_NAME IMAGE_TAG || return 1

  echo "同步镜像：${SRC_HUB}/${IMAGE_NAME}:${IMAGE_TAG} -> ${DST_HUB}/${image_reference}"
  if ! sync_response="$(
    curl --silent --show-error --fail-with-body \
      --get "${API_BASE_URL}${SYNC_API_PATH}" \
      --header "X-API-Key: ${API_KEY}" \
      --data-urlencode "srcHub=${SRC_HUB}" \
      --data-urlencode "dstHub=${DST_HUB}" \
      --data-urlencode "group=${IMAGE_GROUP}" \
      --data-urlencode "image=${IMAGE_NAME}" \
      --data-urlencode "tag=${IMAGE_TAG}"
  )"; then
    echo "镜像同步接口调用失败：" >&2
    print_response "${sync_response}" >&2
    return 1
  fi

  print_response "${sync_response}"
  if [[ ! "${sync_response}" =~ ${sync_code_pattern} ]]; then
    echo "错误：镜像同步未成功" >&2
    return 1
  fi
  if ! sync_status="$(extract_json_string "${sync_response}" status)"; then
    echo "错误：镜像同步响应中缺少 status 字段" >&2
    return 1
  fi
  case "${sync_status}" in
    success | SUCCESS | Success) ;;
    *)
      echo "错误：镜像同步状态为 ${sync_status}" >&2
      return 1
      ;;
  esac

  if ! synced_image="$(extract_json_string "${sync_response}" image)" || [[ -z "${synced_image}" ]]; then
    echo "错误：镜像同步响应中缺少 data.image" >&2
    return 1
  fi
  echo "镜像同步完成：${synced_image}"
}

# watch：持续查询工具状态，直到 ACTIVE、FAILED 或超时。
cmd_watch() {
  local tool_id="${1:-${TOOL_ID}}"
  local timeout="${2:-${TOOL_STATUS_POLL_TIMEOUT}}"
  local started_at="${SECONDS}"
  local elapsed=0
  local tool_status="UNKNOWN"
  local status_response status_reason status_error_code

  if [[ $# -gt 2 ]]; then
    echo "错误：用法：$0 watch [toolId] [超时秒]" >&2
    return 1
  fi

  require_command curl || return 1
  validate_api_key || return 1
  resolve_required "${tool_id}" "请输入要查询的工具 ID（TOOL_ID）" tool_id || return 1
  validate_positive_integer "${TOOL_STATUS_POLL_INTERVAL}" TOOL_STATUS_POLL_INTERVAL || return 1
  validate_positive_integer "${timeout}" "watch 超时" || return 1

  echo "监听工具状态：toolId=${tool_id}，间隔 ${TOOL_STATUS_POLL_INTERVAL}s，超时 ${timeout}s"
  while ((elapsed < timeout)); do
    if ! status_response="$(
      curl --silent --show-error --fail-with-body \
        --get "${API_BASE_URL}${TOOL_STATUS_API_PATH}" \
        --header "X-API-Key: ${API_KEY}" \
        --data-urlencode "toolId=${tool_id}"
    )"; then
      status_error_code="$(extract_json_string "${status_response}" code || true)"
      if [[ "${status_error_code}" == "UNAUTHORIZED" ]]; then
        echo "错误：API Key 鉴权失败，停止监听" >&2
        print_response "${status_response}" >&2
        return 1
      fi
      echo "状态查询失败，将继续重试：" >&2
      print_response "${status_response}" >&2
      sleep "${TOOL_STATUS_POLL_INTERVAL}"
      elapsed=$((SECONDS - started_at))
      continue
    fi

    if ! tool_status="$(extract_json_string "${status_response}" status)"; then
      echo "状态查询响应中缺少 status，将继续重试：" >&2
      print_response "${status_response}" >&2
      sleep "${TOOL_STATUS_POLL_INTERVAL}"
      elapsed=$((SECONDS - started_at))
      continue
    fi

    case "${tool_status}" in
      active | ACTIVE | Active)
        echo "完成：toolId=${tool_id}，status=ACTIVE"
        return 0
        ;;
      failed | FAILED | Failed)
        status_reason="$(extract_json_string "${status_response}" statusReason || true)"
        echo "错误：工具创建失败，toolId=${tool_id}，原因=${status_reason:-未知}" >&2
        return 1
        ;;
      *)
        echo "工具状态：${tool_status}（已等待 ${elapsed}s）"
        ;;
    esac

    sleep "${TOOL_STATUS_POLL_INTERVAL}"
    elapsed=$((SECONDS - started_at))
  done

  echo "错误：等待工具进入 ACTIVE 状态超时，最后状态=${tool_status}" >&2
  return 1
}

# tool：使用指定镜像创建沙箱工具，并自动 watch 创建状态。
cmd_tool() {
  local image_reference="${IMAGE_REFERENCE}"
  local tool_name="${TOOL_NAME}"
  local config_file="${TOOL_CONFIG_FILE}"
  local escaped_tool_name escaped_image tool_payload
  local tool_response tool_id error_code error_message
  local tool_accepted_pattern='"accepted"[[:space:]]*:[[:space:]]*true[[:space:]]*[,}]'

  if [[ "${1:-}" == "--config" ]]; then
    if [[ $# -ne 2 ]]; then
      echo "错误：用法：$0 tool --config <完整请求JSON文件>" >&2
      return 1
    fi
    if [[ -z "${2//[[:space:]]/}" ]]; then
      echo "错误：完整请求 JSON 文件路径不能为空" >&2
      return 1
    fi
    config_file="$2"
  elif [[ -n "${config_file//[[:space:]]/}" ]]; then
    if [[ $# -ne 0 ]]; then
      echo "错误：设置 TOOL_CONFIG_FILE 后不能再传入镜像地址或工具名称" >&2
      return 1
    fi
  else
    if [[ $# -gt 2 ]]; then
      echo "错误：用法：$0 tool [镜像地址] [工具名称]" >&2
      return 1
    fi
    image_reference="${1:-${image_reference}}"
    tool_name="${2:-${tool_name}}"
  fi

  require_command curl || return 1
  validate_api_key || return 1
  validate_positive_integer "${TOOL_STATUS_POLL_INTERVAL}" TOOL_STATUS_POLL_INTERVAL || return 1
  validate_positive_integer "${TOOL_STATUS_POLL_TIMEOUT}" TOOL_STATUS_POLL_TIMEOUT || return 1

  if [[ -n "${config_file//[[:space:]]/}" ]]; then
    if [[ ! -r "${config_file}" ]]; then
      echo "错误：完整请求 JSON 文件不存在或不可读：${config_file}" >&2
      return 1
    fi
    tool_payload="$(<"${config_file}")"
    if [[ -z "${tool_payload//[[:space:]]/}" ]]; then
      echo "错误：完整请求 JSON 文件不能为空：${config_file}" >&2
      return 1
    fi
    if ! tool_name="$(extract_json_string "${tool_payload}" toolName)" || [[ -z "${tool_name}" ]]; then
      echo "错误：完整请求 JSON 中缺少非空 toolName" >&2
      return 1
    fi
    if ! image_reference="$(extract_json_string "${tool_payload}" image)" || [[ -z "${image_reference}" ]]; then
      echo "错误：完整请求 JSON 中缺少非空 customConfiguration.image" >&2
      return 1
    fi
    echo "使用完整接口配置：${config_file}"
  else
    resolve_required "${image_reference}" "请输入用于创建工具的镜像地址（IMAGE_REFERENCE）" image_reference || return 1
    resolve_required "${tool_name}" "请输入要创建的工具名称（TOOL_NAME）" tool_name || return 1

    escaped_tool_name="$(json_escape "${tool_name}")"
    escaped_image="$(json_escape "${image_reference}")"
    printf -v tool_payload \
      '{"toolName":"%s","customConfiguration":{"image":"%s"}}' \
      "${escaped_tool_name}" "${escaped_image}"
  fi

  echo "[1/2] 创建工具：toolName=${tool_name}，image=${image_reference}"
  if ! tool_response="$(
    curl --silent --show-error --fail-with-body \
      --request POST "${API_BASE_URL}${TOOL_API_PATH}" \
      --header "X-API-Key: ${API_KEY}" \
      --header 'Content-Type: application/json' \
      --data "${tool_payload}"
  )"; then
    echo "工具创建接口调用失败：" >&2
    print_response "${tool_response}" >&2
    return 1
  fi

  print_response "${tool_response}"
  if [[ ! "${tool_response}" =~ ${tool_accepted_pattern} ]]; then
    error_code="$(extract_json_string "${tool_response}" errorCode || true)"
    error_message="$(extract_json_string "${tool_response}" errorMessage || true)"
    echo "错误：工具创建未被接受，errorCode=${error_code:-未知}，errorMessage=${error_message:-未知}" >&2
    return 1
  fi
  if ! tool_id="$(extract_json_string "${tool_response}" toolId)" || [[ -z "${tool_id}" ]]; then
    echo "错误：工具创建响应中缺少 toolId" >&2
    return 1
  fi

  echo "工具创建请求已提交：toolId=${tool_id}"
  echo "[2/2] 开始监听工具状态"
  cmd_watch "${tool_id}" "${TOOL_STATUS_POLL_TIMEOUT}"
}

# help：显示子命令、参数和环境变量说明。
print_usage() {
  printf '%s\n' \
    "用法：" \
    "  $0 sync  [目标镜像路径]         同步镜像" \
    "  $0 tool  [镜像地址] [工具名称]  创建沙箱工具并 watch 状态" \
    "  $0 tool  --config <JSON文件>     使用完整接口参数创建并 watch 状态" \
    "  $0 watch [toolId] [超时秒]      查询并持续监听沙箱工具状态" \
    "  $0 help                         显示本帮助" \
    "" \
    "必填参数：" \
    "  命令行未提供时，脚本会进行交互式填写。" \
    "  sync 镜像格式：<IMAGE_GROUP>/<IMAGE_NAME>:<IMAGE_TAG>" \
    "" \
    "可选环境变量：" \
    "  API_BASE_URL=${API_BASE_URL}" \
    "  SRC_HUB=${SRC_HUB}" \
    "  DST_HUB=${DST_HUB}" \
    "  TOOL_STATUS_POLL_INTERVAL=${TOOL_STATUS_POLL_INTERVAL}" \
    "  TOOL_STATUS_POLL_TIMEOUT=${TOOL_STATUS_POLL_TIMEOUT}" \
    "  API_KEY 已在脚本顶部固定配置，不读取环境变量" \
    "" \
    "必填参数也可通过环境变量提供：" \
    "  IMAGE_REFERENCE、TOOL_NAME、TOOL_ID" \
    "  TOOL_CONFIG_FILE 可指定完整创建请求 JSON 文件"
}

case "${1:-help}" in
  sync)
    shift
    cmd_sync "$@"
    ;;
  tool)
    shift
    cmd_tool "$@"
    ;;
  watch)
    shift
    cmd_watch "$@"
    ;;
  help | -h | --help)
    print_usage
    ;;
  *)
    echo "错误：未知命令 $1" >&2
    print_usage >&2
    exit 1
    ;;
esac
