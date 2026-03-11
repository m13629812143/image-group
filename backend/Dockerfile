# 第一阶段：编译
FROM docker.io/library/golang:1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server .

# 第二阶段：运行（体积更小）
FROM docker.io/library/alpine:latest
WORKDIR /app
RUN mkdir -p /app/uploads
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
