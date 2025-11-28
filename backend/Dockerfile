# --- Build Stage ---
    FROM golang:1.25.4-alpine AS builder
    # or: FROM golang:1.25-alpine AS builder でもOK（タグ次第）
    
    WORKDIR /app
    
    # 依存関係のダウンロード
    COPY go.mod go.sum ./
    RUN go mod download
    
    # ソースコードのコピー
    COPY . .
    
    # ビルド (CGO無効化で軽量化)
    RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/main.go
    
    # --- Run Stage ---
    # 実行にはGo本体は不要なので、超軽量なdistrolessイメージを使う
    FROM gcr.io/distroless/static-debian12
    
    WORKDIR /
    
    # ビルド成果物をコピー
    COPY --from=builder /server /server
    
    # サーバー起動
    EXPOSE 8080
    USER nonroot:nonroot
    ENTRYPOINT ["/server"]
    