#!/usr/bin/env bash

set -Eeuo pipefail

# 可通过同名环境变量覆盖以下连接参数。
API_BASE_URL="${API_BASE_URL:-http://pass.myhexin.com/sandbox-adapter}"
SRC_HUB="${SRC_HUB:-hub-dev.hexin.cn:9544}"
DST_HUB="${DST_HUB:-aime-agent-tcr.tencentcloudcr.com}"
IMAGE_GROUP="${IMAGE_GROUP:-ths}"

API_BASE_URL="${API_BASE_URL%/}"

for command_name in curl jq; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "错误：未找到依赖命令 ${command_name}" >&2
    exit 1
  fi
done

read_required() {
  local prompt="$1"
  local variable_name="$2"
  local value

  while true; do
    if ! IFS= read -r -p "${prompt}: " value; then
      echo >&2
      echo "错误：未读取到输入" >&2
      exit 1
    fi
    if [[ -n "${value//[[:space:]]/}" ]]; then
      printf -v "${variable_name}" '%s' "${value}"
      return
    fi
    echo "输入不能为空，请重新输入。" >&2
  done
}

read_required "请输入镜像名称（IMAGE_NAME）" IMAGE_NAME
read_required "请输入镜像标签（IMAGE_TAG）" IMAGE_TAG
read_required "请输入要创建的工具名称（TOOL_NAME）" TOOL_NAME

print_response() {
  local response="$1"

  jq . <<<"${response}" 2>/dev/null || printf '%s\n' "${response}"
}

echo "[1/2] 同步镜像 ${SRC_HUB}/${IMAGE_NAME}:${IMAGE_TAG}"
if ! sync_response="$(
  curl --silent --show-error --fail-with-body \
    --get "${API_BASE_URL}/api/v1/images/sync" \
    --data-urlencode "srcHub=${SRC_HUB}" \
    --data-urlencode "dstHub=${DST_HUB}" \
    --data-urlencode "group=${IMAGE_GROUP}" \
    --data-urlencode "image=${IMAGE_NAME}" \
    --data-urlencode "tag=${IMAGE_TAG}"
)"; then
  echo "镜像同步接口调用失败：" >&2
  print_response "${sync_response}" >&2
  exit 1
fi

print_response "${sync_response}"

if ! jq -e '
  (.code == 0) and
  ((.status // "" | ascii_downcase) == "success") and
  ((.data.image | type) == "string") and
  ((.data.image | length) > 0)
' >/dev/null <<<"${sync_response}"; then
  echo "错误：镜像同步未成功，停止创建工具" >&2
  exit 1
fi

synced_image="$(jq -r '.data.image' <<<"${sync_response}")"
echo "同步后的镜像地址：${synced_image}"

tool_payload="$(
  jq -n \
    --arg tool_name "${TOOL_NAME}" \
    --arg image "${synced_image}" \
    '{
      toolName: $tool_name,
      customConfiguration: {
        image: $image
      }
    }'
)"

echo "[2/2] 创建工具 ${TOOL_NAME}"
if ! tool_response="$(
  curl --silent --show-error --fail-with-body \
    --request POST "${API_BASE_URL}/api/v1/sandbox/tools" \
    --header 'Content-Type: application/json' \
    --data "${tool_payload}"
)"; then
  echo "工具创建接口调用失败：" >&2
  print_response "${tool_response}" >&2
  exit 1
fi

print_response "${tool_response}"

if ! jq -e '
  (.accepted == true) and
  ((.status // "" | ascii_upcase) != "FAILED")
' >/dev/null <<<"${tool_response}"; then
  echo "错误：工具创建未被接受或最终创建失败" >&2
  exit 1
fi

tool_id="$(jq -r '.toolId // empty' <<<"${tool_response}")"
tool_status="$(jq -r '.status // "UNKNOWN"' <<<"${tool_response}")"
echo "完成：toolId=${tool_id:-N/A}，status=${tool_status}"
