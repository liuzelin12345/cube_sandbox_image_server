#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# CubeSandbox PaaS 镜像模板工具 (无 jq 依赖)
#
# 用法:
#   ./cubesandbox-cli.sh create <镜像地址> [选项]            # 创建模板并 watch 构建
#   ./cubesandbox-cli.sh redo      <templateID> --node <N>... # 定向分发到指定节点(扩容用)
#   ./cubesandbox-cli.sh sync-node <nodeID或IP>              # 将所有模板同步到指定节点(扩容用)
#   ./cubesandbox-cli.sh build     <templateID>              # 全节点重建模板(RedoModeAll)
#   ./cubesandbox-cli.sh watch     <templateID> <buildID>    # 轮询构建状态
#
# 环境变量:
#   CUBEAPI_BASE       - CubeAPI 基地址 (默认走 PaaS 网关; 直连示例见下)
#   PAAS_CLUSTER_CODE  - PaaS 集群代码 (默认 beimeipri; 直连 CubeAPI 时设为空)
#   REGISTRY_USER      - 私有镜像仓库用户名
#   REGISTRY_PASS      - 私有镜像仓库密码
#
# 自定义 CUBEAPI_BASE 示例:
#   # 直连某台 CubeAPI 的 3000 端口(无需 PaaS 集群参数):
#   CUBEAPI_BASE=http://10.0.0.49:3000/cubeapi/v1 PAAS_CLUSTER_CODE= \
#     ./cubesandbox-cli.sh sync-node 10.0.0.59
#
#   # 走另一个 PaaS 集群:
#   PAAS_CLUSTER_CODE=other ./cubesandbox-cli.sh build tpl-xxx

set -euo pipefail

# CUBEAPI_BASE 可被环境变量覆盖; 未设置时用 PaaS 网关默认值。
CUBEAPI_BASE="${CUBEAPI_BASE:-https://paas.myhexin.com/cubesandbox}"

# PAAS_CLUSTER_CODE 用 ${VAR-default}: 未设置时给默认值, 显式设为空则保持空
# (直连 CubeAPI 时设为空即可去掉 ?paasBasicClusterCode 查询参数)。
PAAS_CLUSTER_CODE="${PAAS_CLUSTER_CODE-beimeipri}"
if [[ -n "$PAAS_CLUSTER_CODE" ]]; then
  PAAS_QS="paasBasicClusterCode=${PAAS_CLUSTER_CODE}"
else
  PAAS_QS=""
fi

RED='\033[0;31m'; GREEN='\033[0;32m'; BLUE='\033[0;34m'; YELLOW='\033[0;33m'; NC='\033[0m'
info() { echo -e "${BLUE}[INFO]${NC} $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}   $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*" >&2; }
err()  { echo -e "${RED}[ERR]${NC}  $*" >&2; }

# 拼接 URL 与查询串: url "/templates/tpl-x/builds/b/status" → 自动附加 PAAS_QS(若有)
url() {
  if [[ -n "$PAAS_QS" ]]; then
    echo "${CUBEAPI_BASE}$1?${PAAS_QS}"
  else
    echo "${CUBEAPI_BASE}$1"
  fi
}

# 从 JSON 中提取字符串值: json_get '{"a":"hello"}' a → hello
json_get() {
  echo "$1" | grep -oP "\"$2\"\s*:\s*\"[^\"]*\"" | head -1 | sed 's/.*"'"$2"'" *: *"//;s/"$//'
}

# 从 JSON 中提取数字值: json_get_num '{"a":42}' a → 42
json_get_num() {
  echo "$1" | grep -oP "\"$2\"\s*:\s*[0-9]+" | head -1 | sed 's/.*"'"$2"'" *: *//'
}

# 从模板列表 JSON 中提取所有 templateID(无 jq 依赖)。
json_template_ids() {
  echo "$1" | grep -oP '"templateID"\s*:\s*"[^"]+"' | sed 's/.*"templateID" *: *"//;s/"$//'
}

check_ret() {
  local code
  code=$(json_get "$1" "ret_code")
  if [[ -n "$code" && "$code" != "200" ]]; then
    local msg
    msg=$(json_get "$1" "ret_msg")
    err "API 错误 (code=$code): ${msg:-unknown}"; return 1
  fi
}

# ─── watch ───────────────────────────────────────────────────────────────────

cmd_watch() {
  local tpl="${1:?用法: watch <templateID> <buildID>}"
  local build="${2:?用法: watch <templateID> <buildID>}"
  local timeout="${3:-600}"
  local elapsed=0 spinner='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏' si=0

  info "监控构建: tpl=${tpl}  build=${build}  超时=${timeout}s"
  echo ""

  while (( elapsed < timeout )); do
    local resp
    resp=$(curl -sf "$(url "/templates/${tpl}/builds/${build}/status")" 2>/dev/null) || {
      sleep 5; elapsed=$((elapsed + 5)); continue
    }

    local status progress message
    status=$(json_get "$resp" "status")
    progress=$(json_get_num "$resp" "progress")
    message=$(json_get "$resp" "message")
    status="${status:-unknown}"
    progress="${progress:-0}"

    local filled=$(( progress * 30 / 100 ))
    local bar
    bar=$(printf "%${filled}s" | tr ' ' '█')$(printf "%$((30 - filled))s" | tr ' ' '░')
    printf "\r  ${spinner:$((si % ${#spinner})):1} [%s] %3d%%  %-20s  %s" "$bar" "$progress" "$status" "$message"
    si=$((si + 1))

    [[ "$status" == "ready" ]] && printf "\n\n" && ok "✅ 模板构建成功: ${tpl}" && return 0
    [[ "$status" == "error" ]] && printf "\n\n" && err "❌ 构建失败: ${message}" && return 1

    sleep 5; elapsed=$((elapsed + 5))
  done

  printf "\n"; err "⏰ 超时 (${timeout}s)"; return 1
}

# ─── create ───────────────────────────────────────────────────────────────────

cmd_create() {
  local image="" writable_layer="1G" probe_port=""
  local cpu="500" memory="2048"
  local -a expose_ports=()

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --writable-layer-size) writable_layer="$2"; shift 2 ;;
      --expose-port)         expose_ports+=("$2"); shift 2 ;;
      --probe)               probe_port="$2";      shift 2 ;;
      --cpu)                 cpu="$2";             shift 2 ;;
      --memory)              memory="$2";          shift 2 ;;
      -h|--help)
        echo "用法: create <镜像地址> [--cpu N] [--memory N] [--expose-port PORT]... [--probe PORT] [--writable-layer-size 1G]"
        return 0 ;;
      -*) err "未知选项: $1"; return 1 ;;
      *)  image="$1"; shift ;;
    esac
  done
  [[ -z "$image" ]] && err "请指定镜像地址" && return 1
  [[ ${#expose_ports[@]} -eq 0 ]] && err "请至少指定一个 --expose-port" && return 1
  [[ -z "$probe_port" ]] && probe_port="${expose_ports[0]}"

  # 拼装 ports JSON 数组
  local ports_json="["
  for i in "${!expose_ports[@]}"; do
    [[ $i -gt 0 ]] && ports_json+=","
    ports_json+="${expose_ports[$i]}"
  done
  ports_json+="]"

  # 手拼 JSON body（不依赖 jq）
  local body="{\"image\":\"${image}\",\"writableLayerSize\":\"${writable_layer}\",\"exposedPorts\":${ports_json},\"probePort\":${probe_port},\"probePath\":\"/health\",\"cpu\":${cpu},\"memory\":${memory}"

  if [[ -n "${REGISTRY_USER:-}" && -n "${REGISTRY_PASS:-}" ]]; then
    body+=",\"registryUsername\":\"${REGISTRY_USER}\",\"registryPassword\":\"${REGISTRY_PASS}\""
  fi
  body+="}"

  info "创建模板: ${image}"
  info "CPU=${cpu}m  MEM=${memory}MiB  可写层=${writable_layer}  端口=${ports_json}  探针=${probe_port}"
  echo ""

  local resp
  resp=$(curl -sf -X POST "$(url "/templates")" \
    -H 'Content-Type: application/json' \
    -d "$body") || { err "请求失败"; return 1; }

  check_ret "$resp" || return 1

  local tpl_id build_id
  tpl_id=$(json_get "$resp" "templateID")
  build_id=$(json_get "$resp" "jobID")

  ok "已提交: template=${tpl_id}  build=${build_id}"
  echo ""
  cmd_watch "$tpl_id" "$build_id"
}

# ─── 提交 redo 并(可选)watch ──────────────────────────────────────────────────
# 统一处理 redo/build: 都打到 POST /templates/{id} (转发 CubeMaster /cube/template/redo)
# $1=templateID  $2=JSON body  $3=场景描述  $4=是否 watch(true/false)
submit_redo() {
  local tpl="$1" body="$2" desc="$3" do_watch="$4"

  info "${desc}: template=${tpl}"
  info "请求体: ${body}"
  echo ""

  local resp
  resp=$(curl -sf -X POST "$(url "/templates/${tpl}")" \
    -H 'Content-Type: application/json' \
    -d "$body") || { err "请求失败"; return 1; }

  check_ret "$resp" || return 1

  local build_id
  build_id=$(json_get "$resp" "jobID")
  [[ -z "$build_id" ]] && err "未拿到 jobID, 响应: ${resp}" && return 1

  ok "已提交: template=${tpl}  build=${build_id}"
  echo ""

  if [[ "$do_watch" == "true" ]]; then
    cmd_watch "$tpl" "$build_id"
  else
    info "跳过 watch。可手动查询: $0 watch ${tpl} ${build_id}"
  fi
}

# ─── redo (定向分发到指定节点; 扩容场景) ───────────────────────────────────────────────────────────────────

cmd_redo() {
  local tpl="" failed_only=false do_watch=true
  local -a nodes=()

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --node)        nodes+=("$2"); shift 2 ;;
      --failed-only) failed_only=true; shift ;;
      --no-watch)    do_watch=false; shift ;;
      -h|--help)
        echo "用法: redo <templateID> --node <节点ID或IP>... [--failed-only] [--no-watch]"
        echo "说明: 仅把模板重新分发到指定节点(扩容用); 不改动其它节点。"
        echo "      如需全节点重建请用 'build'。"
        return 0 ;;
      -*) err "未知选项: $1"; return 1 ;;
      *)  tpl="$1"; shift ;;
    esac
  done

  [[ -z "$tpl" ]] && err "请指定 templateID" && return 1
  # 安全护栏: 防止空 body 误触 RedoModeAll(全节点重建)。定向 redo 必须给条件。
  if [[ ${#nodes[@]} -eq 0 && "$failed_only" != "true" ]]; then
    err "redo 需要 --node <节点> (或 --failed-only)。"
    err "若确实要全节点重建, 请改用: $0 build ${tpl}"
    return 1
  fi

  # 拼 body: distribution_scope + failed_only (透传给 CubeMaster, 用 snake_case)
  local body="{" first=true
  if [[ ${#nodes[@]} -gt 0 ]]; then
    local arr="["
    for i in "${!nodes[@]}"; do
      [[ $i -gt 0 ]] && arr+=","
      arr+="\"${nodes[$i]}\""
    done
    arr+="]"
    body+="\"distribution_scope\":${arr}"
    first=false
  fi
  if [[ "$failed_only" == "true" ]]; then
    [[ "$first" == "false" ]] && body+=","
    body+="\"failed_only\":true"
  fi
  body+="}"

  submit_redo "$tpl" "$body" "定向分发(redo)" "$do_watch"
}

# ─── sync-node (将所有模板同步到指定节点; 扩容场景) ───────────────────────────────────────────────────────────────────

cmd_sync_node() {
  local node="" do_watch=false include_non_ready=false limit=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --node)              node="$2"; shift 2 ;;
      --watch)             do_watch=true; shift ;;
      --include-non-ready) include_non_ready=true; shift ;;
      --limit)             limit="$2"; shift 2 ;;
      -h|--help)
        echo "用法: sync-node <节点ID或IP> [--watch] [--include-non-ready] [--limit N]"
        echo "说明: 列出 CubeAPI 中所有模板, 并逐个 redo 到指定节点。扩容后给新节点补齐模板用。"
        echo "      默认只同步 status=READY 的模板; 默认不逐个 watch, 只提交 job。"
        return 0 ;;
      -*) err "未知选项: $1"; return 1 ;;
      *)  node="$1"; shift ;;
    esac
  done

  [[ -z "$node" ]] && err "请指定节点ID或IP" && return 1

  info "获取模板列表: $(url "/templates")"
  local list_resp
  list_resp=$(curl -sf "$(url "/templates")") || { err "获取模板列表失败"; return 1; }
  check_ret "$list_resp" || return 1

  # CubeAPI 返回字段是 templateID/status。为了避免把 failed/building 的模板也分发,
  # 默认只取 status=READY 附近的 templateID。这里用轻量文本解析,不依赖 jq。
  local ids=()
  if [[ "$include_non_ready" == "true" ]]; then
    while IFS= read -r id; do
      [[ -n "$id" ]] && ids+=("$id")
    done < <(json_template_ids "$list_resp")
  else
    # 将每个对象压成一行,过滤包含 "status":"READY" 的对象,再提取 templateID。
    while IFS= read -r id; do
      [[ -n "$id" ]] && ids+=("$id")
    done < <(echo "$list_resp" \
      | tr '\n' ' ' \
      | sed 's/},{/}\n{/g' \
      | grep -E '"status"[[:space:]]*:[[:space:]]*"READY"' \
      | grep -oP '"templateID"\s*:\s*"[^"]+"' \
      | sed 's/.*"templateID" *: *"//;s/"$//')
  fi

  if [[ -n "$limit" && "$limit" =~ ^[0-9]+$ ]]; then
    ids=("${ids[@]:0:$limit}")
  fi

  [[ ${#ids[@]} -eq 0 ]] && warn "没有找到可同步的模板" && return 0

  info "准备同步 ${#ids[@]} 个模板到节点 ${node}"
  warn "这会为每个模板提交一个 redo job, 仅影响节点 ${node}, 不会全节点重建。"
  echo ""

  local ok_count=0 fail_count=0
  local -a failed=()
  for tpl in "${ids[@]}"; do
    echo "------------------------------------------------------------"
    if submit_redo "$tpl" "{\"distribution_scope\":[\"${node}\"]}" "同步到节点 ${node}" "$do_watch"; then
      ok_count=$((ok_count + 1))
    else
      fail_count=$((fail_count + 1))
      failed+=("$tpl")
      warn "模板 ${tpl} 提交失败, 继续处理下一个"
    fi
  done

  echo "------------------------------------------------------------"
  ok "sync-node 完成: 成功提交 ${ok_count}, 失败 ${fail_count}, 目标节点 ${node}"
  if [[ ${#failed[@]} -gt 0 ]]; then
    warn "失败模板: ${failed[*]}"
    return 1
  fi
}

# ─── build (全节点重建; RedoModeAll) ──────────────────────────────────────────

cmd_build() {
  local tpl="" do_watch=true force=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --no-watch) do_watch=false; shift ;;
      -y|--yes)   force=true; shift ;;
      -h|--help)
        echo "用法: build <templateID> [-y] [--no-watch]"
        echo "说明: 对所有健康节点重新分发/重建快照(RedoModeAll)。会扰动在役节点, 请谨慎。"
        echo "      仅想给新增节点补模板请用 'sync-node'。"
        return 0 ;;
      -*) err "未知选项: $1"; return 1 ;;
      *)  tpl="$1"; shift ;;
    esac
  done

  [[ -z "$tpl" ]] && err "请指定 templateID" && return 1

  warn "build 会对【所有健康节点】重新分发并重建快照 (RedoModeAll)。"
  if [[ "$force" != "true" ]]; then
    read -r -p "确认继续? 输入 yes 继续: " ans
    [[ "$ans" != "yes" ]] && err "已取消" && return 1
  fi

  # 空 body → CubeMaster RedoModeAll
  submit_redo "$tpl" "{}" "全节点重建(build)" "$do_watch"
}

# ─── 入口 ─────────────────────────────────────────────────────────────────────

case "${1:-}" in
  create)    shift; cmd_create    "$@" ;;
  redo)      shift; cmd_redo      "$@" ;;
  sync-node) shift; cmd_sync_node "$@" ;;
  build)     shift; cmd_build     "$@" ;;
  watch)     shift; cmd_watch     "$@" ;;
  *) cat <<EOF
用法:
  $0 create <镜像地址> [--cpu N] [--memory N] [--expose-port PORT]... [--probe PORT]
  $0 redo   <templateID> --node <节点ID或IP>... [--failed-only] [--no-watch]
  $0 sync-node <节点ID或IP> [--watch] [--include-non-ready] [--limit N]
  $0 build  <templateID> [-y] [--no-watch]
  $0 watch  <templateID> <buildID> [超时秒]

环境变量:
  CUBEAPI_BASE       CubeAPI 基地址 (默认 ${CUBEAPI_BASE})
  PAAS_CLUSTER_CODE  PaaS 集群代码 (默认 beimeipri; 直连 CubeAPI 时设为空字符串)
  REGISTRY_USER / REGISTRY_PASS  私有镜像仓库凭据
EOF
  ;;
esac
