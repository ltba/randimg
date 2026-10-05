# Build stage — CGO_ENABLED=0: glebarez/sqlite 为 pure Go 驱动, 产物静态链接.
FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o randimg ./cmd/server

# Runtime stage — distroless/static 自带 CA 证书 (出站 https 拉图源), nonroot 以 UID 65532 运行.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /build/randimg .
COPY --from=builder /build/static ./static

# data 目录由程序启动时自动创建; 卷挂载点权限由部署侧保证.

EXPOSE 8080

USER nonroot

ENTRYPOINT ["./randimg"]
