#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DOCKERFILE="$ROOT_DIR/sf-dockerfile"
PLATFORM="linux/amd64"

read_version() {
  sed -n '1p' "$ROOT_DIR/VERSION" | tr -d '[:space:]'
}

usage() {
  cat <<'EOF'
用法:
  ./scripts/build-sf-docker-amd64.sh [--image <镜像名>] [--tag <标签>] [--push] [--builder <builder>] [--retries <次数>]

示例:
  ./scripts/build-sf-docker-amd64.sh
  ./scripts/build-sf-docker-amd64.sh --image linghuajian1/new-api
  ./scripts/build-sf-docker-amd64.sh --image linghuajian1/new-api --tag v0.0.0-amd64 --push
  ./scripts/build-sf-docker-amd64.sh --image docker.io/linghuajian1/new-api --push

说明:
  - 默认平台固定为 linux/amd64
  - 默认镜像名为 linghuajian1/new-api
  - 不带 --push 时会使用 --load，把镜像加载到本地 Docker
  - 带 --push 时会使用 --push，要求你已先执行 docker login
  - 推送模式默认失败重试 3 次，用 --retries 0 可关闭
EOF
}

VERSION=$(read_version)
IMAGE_NAME="linghuajian1/new-api"
IMAGE_TAG="${VERSION:-latest}-amd64"
PUSH="false"
BUILDER=""
RETRIES=3
RETRY_DELAY=15

while [ $# -gt 0 ]; do
  case "$1" in
    --image|-i)
      [ $# -ge 2 ] || { echo '缺少 --image 的值' >&2; exit 1; }
      IMAGE_NAME="$2"
      shift 2
      ;;
    --tag|-t)
      [ $# -ge 2 ] || { echo '缺少 --tag 的值' >&2; exit 1; }
      IMAGE_TAG="$2"
      shift 2
      ;;
    --builder|-b)
      [ $# -ge 2 ] || { echo '缺少 --builder 的值' >&2; exit 1; }
      BUILDER="$2"
      shift 2
      ;;
    --retries)
      [ $# -ge 2 ] || { echo '缺少 --retries 的值' >&2; exit 1; }
      RETRIES="$2"
      shift 2
      ;;
    --retry-delay)
      [ $# -ge 2 ] || { echo '缺少 --retry-delay 的值' >&2; exit 1; }
      RETRY_DELAY="$2"
      shift 2
      ;;
    --push)
      PUSH="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "未知参数: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo '未找到 docker 命令' >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo 'Docker daemon 不可用，请先启动 Docker' >&2
  exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
  echo '当前 Docker 未启用 buildx' >&2
  exit 1
fi

if [ ! -f "$DOCKERFILE" ]; then
  echo "未找到 Dockerfile: $DOCKERFILE" >&2
  exit 1
fi

case "$RETRIES" in
  ''|*[!0-9]*)
    echo '--retries 必须是非负整数' >&2
    exit 1
    ;;
esac

case "$RETRY_DELAY" in
  ''|*[!0-9]*)
    echo '--retry-delay 必须是非负整数秒' >&2
    exit 1
    ;;
esac

if [ "$PUSH" = "true" ]; then
  case "$IMAGE_NAME" in
    */*) ;;
    *)
      echo '启用 --push 时，请使用带命名空间或仓库地址的镜像名，例如 linghuajian1/new-api' >&2
      exit 1
      ;;
  esac
fi

IMAGE_REF="${IMAGE_NAME}:${IMAGE_TAG}"
CMD=(docker buildx build --platform "$PLATFORM" -f "$DOCKERFILE" -t "$IMAGE_REF")

if [ -n "$BUILDER" ]; then
  CMD+=(--builder "$BUILDER")
fi

if [ "$PUSH" = "true" ]; then
  CMD+=(--push)
else
  CMD+=(--load)
fi

CMD+=("$ROOT_DIR")

echo "镜像: $IMAGE_REF"
echo "平台: $PLATFORM"
if [ "$PUSH" = "true" ]; then
  echo '模式: 构建并推送'
  echo "推送失败重试: $RETRIES 次"
else
  echo '模式: 构建并加载到本地 Docker'
fi

attempt=0
max_attempts=1
if [ "$PUSH" = "true" ]; then
  max_attempts=$((RETRIES + 1))
fi

while :; do
  attempt=$((attempt + 1))
  if "${CMD[@]}"; then
    break
  fi

  if [ "$attempt" -ge "$max_attempts" ]; then
    echo "失败: $IMAGE_REF" >&2
    exit 1
  fi

  echo "推送失败，${RETRY_DELAY}s 后重试 (${attempt}/${max_attempts})..." >&2
  sleep "$RETRY_DELAY"
done

echo "完成: $IMAGE_REF"
