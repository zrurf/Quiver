# 构建阶段
FROM golang:1.25.6-alpine3.23 AS builder

# 安装编译依赖
RUN apk add --no-cache \
    gcc \
    musl-dev \
    git \
    make

WORKDIR /app

# 启用cgo并设置优化标志
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    GOGC=off \
    GOPROXY=https://goproxy.cn,direct

# 复制go模块文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译（UDP服务需要网络优化）
RUN go build -tags=sonic,netgo \
    -ldflags="-s -w -linkmode=external -extldflags=-static" \
    -o game_gateway .

# 运行时阶段
FROM alpine:3.23

# 安装最小运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 设置网络参数（优化UDP性能）
RUN echo "net.core.rmem_max=26214400" >> /etc/sysctl.conf && \
    echo "net.core.wmem_max=26214400" >> /etc/sysctl.conf && \
    echo "net.ipv4.udp_mem=4096 87380 16777216" >> /etc/sysctl.conf

WORKDIR /app

# 复制二进制文件
COPY --from=builder /app/game_gateway .

# 创建运行用户
RUN adduser -D -u 1000 appuser && \
    chown -R appuser:appuser /app

USER appuser

# 启动命令
CMD ["./game_gateway"]