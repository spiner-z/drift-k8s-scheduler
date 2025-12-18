# Usage:
#   docker build -t crater-harbor.act.buaa.edu.cn/user-zhangry/drift-k8s-scheduler:v1 .

# 使用官方Go镜像进行编译
FROM crater-harbor.act.buaa.edu.cn/docker.io/golang:1.22
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 禁用CGO，构建Linux静态二进制
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o drift-scheduler .

# 使用distroless镜像作为运行环境
FROM crater-harbor.act.buaa.edu.cn/gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/drift-scheduler .
USER nonroot:nonroot
ENTRYPOINT ["/drift-scheduler", "--config=/etc/kubernetes/scheduler-config.yaml"]
