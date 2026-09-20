# Echoes 容器镜像：多阶段构建，产出 CGO_ENABLED=0 静态二进制。
# 运行参数通过命令行传入，例如：
#   docker run -d -p 9966:9966 ghcr.io/mutantcat-working-group/echoes:latest -server_name=web-01 -port=9966
FROM golang:1.25-alpine AS build

ARG VERSION=1.0.20260920

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/echoes .

FROM alpine:3.21

RUN adduser -D -H -u 10001 echoes

COPY --from=build /out/echoes /usr/local/bin/echoes

USER echoes
EXPOSE 9966

ENTRYPOINT ["/usr/local/bin/echoes"]
