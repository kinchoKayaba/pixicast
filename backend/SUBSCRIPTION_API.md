# 購読登録 API ドキュメント

## 概要

ユーザーが YouTube チャンネルを購読登録するための REST API です。

## エンドポイント

```
POST /v1/subscriptions
```

## リクエスト

### ヘッダー

```
Content-Type: application/json
```

### ボディ

```json
{
  "platform": "youtube",
  "input": "<URL or @handle or UCxxx...>"
}
```

### 入力フォーマット

以下の 3 つの形式をサポート：

1. **YouTube URL**

   ```json
   {"platform": "youtube", "input": "https://www.youtube.com/@junchannel"}
   {"platform": "youtube", "input": "https://www.youtube.com/channel/UCxxxxxxxxxxxx"}
   {"platform": "youtube", "input": "https://www.youtube.com/@junchannel/featured"}
   ```

2. **@handle**

   ```json
   { "platform": "youtube", "input": "@junchannel" }
   ```

3. **Channel ID (UCxxx...)**
   ```json
   { "platform": "youtube", "input": "UCxxxxxxxxxxxx" }
   ```

## レスポンス

### 成功 (201 Created)

```json
{
  "subscription": {
    "user_id": 1,
    "platform": "youtube",
    "source_id": "uuid-here",
    "channel_id": "UCxxxxxxxxxxxx",
    "handle": "junchannel",
    "display_name": "Jun Channel",
    "thumbnail_url": "https://...",
    "enabled": true
  }
}
```

### エラー

#### 400 Bad Request

```json
{"error": "invalid JSON"}
{"error": "only youtube platform is supported"}
{"error": "input is required"}
{"error": "invalid input format"}
```

#### 404 Not Found

```json
{"error": "channel not found for handle: @xxx"}
{"error": "channel not found"}
```

#### 500 Internal Server Error

```json
{ "error": "failed to create subscription" }
```

## 動作

1. **入力正規化**: URL/handle/channelID を解析
2. **チャンネル解決**: @handle の場合は YouTube Data API v3 で channelID に解決
3. **チャンネル情報取得**: チャンネルの詳細情報を取得
4. **DB 保存**: `sources` と `user_subscriptions` テーブルに upsert
5. **非同期取り込み**: goroutine で EnqueueIngest 呼び出し（現在はスタブ）
6. **レスポンス返却**: 201 Created で購読情報を返す

## 冪等性

- 同じチャンネルを複数回登録しても成功（upsert）
- `enabled=true` で更新される

## 認証

現在は `user_id=1` 固定。将来 JWT 等で実装予定。

## テスト方法

### 前提条件

1. DB スキーマを適用:

   ```bash
   cd backend
   psql $DATABASE_URL < sql/schema.sql
   ```

2. 環境変数を設定:

   ```bash
   export YOUTUBE_API_KEY="your-api-key"
   export DATABASE_URL="postgresql://..."
   ```

3. サーバー起動:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

### curl でテスト

#### 1. URL 形式（@handle）

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "youtube",
    "input": "https://www.youtube.com/@junchannel"
  }'
```

#### 2. @handle 形式

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "youtube",
    "input": "@junchannel"
  }'
```

#### 3. Channel ID 形式

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "youtube",
    "input": "UCxxxxxxxxxxxx"
  }'
```

### エラーケースのテスト

#### 無効な handle

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "youtube",
    "input": "@nonexistentchannel999999"
  }'
# Expected: 404 Not Found
```

#### 無効な platform

```bash
curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "twitch",
    "input": "@somechannel"
  }'
# Expected: 400 Bad Request
```

## DB スキーマ

### sources テーブル

```sql
CREATE TABLE sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id STRING NOT NULL,
    external_id STRING NOT NULL,
    handle STRING,
    display_name STRING,
    thumbnail_url STRING,
    uploads_playlist_id STRING,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (platform_id, external_id)
);
```

### user_subscriptions テーブル

```sql
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INT NOT NULL,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    enabled BOOL NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (user_id, source_id)
);
```

## 今後の拡張

1. **認証**: JWT トークンから user_id を取得
2. **非同期取り込み**: Cloud Tasks / PubSub で動画取り込みジョブを実行
3. **購読一覧取得**: `GET /v1/subscriptions` エンドポイント追加
4. **購読解除**: `DELETE /v1/subscriptions/:id` エンドポイント追加
5. **ページネーション**: 大量の購読に対応
6. **レート制限**: YouTube API クォータ管理
