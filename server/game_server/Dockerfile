# 构建阶段
FROM golang:1.25.6-alpine3.23 AS builder

# 安装编译依赖
RUN apk add --no-cache \
    gcc \
    musl-dev \
    git \
    make

WORKDIR /app

# 设置环境变量
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

# 复制go模块文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN go build -tags=sonic \
    -ldflags="-s -w" \
    -o game_server .

# 运行时阶段
FROM alpine:3.23

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    postgresql-client

# 设置时区
ENV TZ=Asia/Shanghai \
    GIN_MODE=release

WORKDIR /app

# 复制二进制文件
COPY --from=builder /app/game_server .

# 创建非root用户
RUN addgroup -g 1000 app && \
    adduser -u 1000 -G app -D app && \
    chown -R app:app /app

USER app

# 启动命令
CMD ["./game_server"]