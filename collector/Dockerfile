# Stage 1: build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o collector .

# Stage 2: runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/collector .
RUN mkdir -p /data
VOLUME ["/data"]
ENV ETCD_URL=http://etcd:2379
ENV WORKER_ID=0
ENTRYPOINT ["./collector"]
CMD ["--mode", "worker"]
