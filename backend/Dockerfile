# ビルドステージ
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# 依存関係をキャッシュ
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# バイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

# 実行ステージ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# ビルドしたバイナリをコピー
COPY --from=builder /app/server .

# ポート8080を公開
EXPOSE 8080

# サーバーを起動
CMD ["./server"]
