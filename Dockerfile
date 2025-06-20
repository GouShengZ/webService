# 使用官方Go镜像作为构建环境
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包，并在同一层清理缓存
RUN apk add --no-cache git ca-certificates tzdata && \
    apk upgrade --no-cache

# 设置环境变量
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 复制go mod文件
COPY go.mod go.sum ./

# 设置Go代理（如果在中国，可以提高下载速度）
ENV GOPROXY=https://goproxy.cn,direct

# 下载依赖
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 构建应用
RUN go build -ldflags="-w -s" -o main .

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:3.19

# 安装ca-certificates、tzdata和curl用于健康检查，并清理缓存
RUN apk --no-cache add ca-certificates tzdata curl && \
    apk upgrade --no-cache && \
    rm -rf /var/cache/apk/*

# 设置时区
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo 'Asia/Shanghai' > /etc/timezone

# 创建非root用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 创建日志目录
RUN mkdir -p /app/logs && chown -R appuser:appgroup /app

# 从构建阶段复制二进制文件
COPY --from=builder --chown=appuser:appgroup /app/main .

# 复制配置文件
COPY --from=builder --chown=appuser:appgroup /app/config.yaml .

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查 - 使用curl进行健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 运行应用
CMD ["./main"]