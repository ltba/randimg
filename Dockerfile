# Build stage — 固定在构建机原生平台, Go 交叉编译产出目标架构二进制 (CGO=0), 不经 QEMU.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

WORKDIR /build

COPY go.mod go.sum ./
# mod 与 build cache 跨构建持久 (buildkit cache mount), 版本注入不再使编译全量重跑.
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X main.version=${VERSION}" -o randimg ./cmd/server

# Runtime stage — distroless/static 自带 CA 证书 (出站 https 拉图源), nonroot 以 UID 65532 运行.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /build/randimg .
COPY --from=builder /build/static ./static

# data 目录由程序启动时自动创建; 卷挂载点权限由部署侧保证.

EXPOSE 8080

USER nonroot

ENTRYPOINT ["./randimg"]
