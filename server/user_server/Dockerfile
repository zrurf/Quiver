# 构建阶段
FROM golang:1.26.0-alpine3.23 AS builder

# 安装编译依赖（sonic需要cgo和gcc）
RUN apk add --no-cache \
    gcc \
    musl-dev \
    git \
    make

# 设置工作目录
WORKDIR /app

# 启用cgo（sonic需要）
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

# 复制go模块文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译（使用sonic标签）
RUN go build -tags=sonic -ldflags="-s -w" -o user_server .

# 运行时阶段
FROM alpine:3.23

# 安装必要的运行时库
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    libc6-compat

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN addgroup -g 1000 app && \
    adduser -u 1000 -G app -D app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder --chown=app:app /app/user_server .

# 切换到非root用户
USER app

# 启动命令
CMD ["./user_server"]