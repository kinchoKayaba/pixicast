# Pixicast スキーマ移行ガイド

## 概要

既存の `programs` + `sources` + `user_subscriptions` スキーマから、
新しい `platforms` + `sources` + `user_subscriptions` + `events` スキーマへの移行ガイドです。

## 主な変更点

### 1. テーブル構造の変更

#### 新規追加

- **platforms**: プラットフォームマスタテーブル
- **events**: `programs` を置き換える正規化されたタイムライン項目テーブル

#### 変更

- **sources**:

  - `last_fetched_at`, `fetch_status` カラム追加
  - より詳細な状態管理が可能に

- **user_subscriptions**:
  - `user_id` の型が `INT` → `BIGINT` に変更
  - `priority` カラム追加（表示順序制御用）
  - PRIMARY KEY が `(user_id, source_id)` に変更（複合主キー）

#### 削除

- **programs**: `events` テーブルに統合

### 2. データモデルの改善

#### programs → events の変更点

| 旧 (programs)          | 新 (events)              | 備考                                   |
| ---------------------- | ------------------------ | -------------------------------------- |
| platform_name (string) | platform_id (text FK)    | 正規化：platforms テーブルを参照       |
| -                      | source_id (uuid FK)      | 追加：どのチャンネルのイベントか       |
| -                      | external_event_id (text) | 追加：YouTube videoId 等               |
| -                      | type (text)              | 追加：live/scheduled/video/premiere    |
| link_url               | url                      | 名前変更                               |
| -                      | metrics (jsonb)          | 追加：統計情報（views, likes 等）      |
| -                      | UNIQUE 制約              | 追加：(platform_id, external_event_id) |

### 3. sqlc クエリの変更

#### 旧スキーマのクエリ

```sql
-- query.sql
-- name: ListPrograms :many
SELECT * FROM programs ORDER BY start_at ASC;

-- name: UpsertSource :one
INSERT INTO sources (platform_id, external_id, ...) ...
```

#### 新スキーマのクエリ

```sql
-- queries/query_timeline.sql
-- name: ListTimeline :many
SELECT e.*, s.display_name, s.thumbnail_url, s.handle
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE us.user_id = $1 AND us.enabled = true
ORDER BY COALESCE(e.start_at, e.published_at) DESC
LIMIT $3;
```

## 移行手順

### ステップ 1: バックアップ

```bash
# PostgreSQL
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql

# CockroachDB
cockroach dump pixicast --url "$DATABASE_URL" > backup_$(date +%Y%m%d).sql
```

### ステップ 2: 新スキーマ適用

```bash
cd backend

# 1. 新テーブル作成
psql $DATABASE_URL < sql/migrations/001_create_tables.sql

# 2. プラットフォームマスタ投入
psql $DATABASE_URL < sql/migrations/002_seed_platforms.sql
```

### ステップ 3: データ移行（必要な場合）

既存の `programs` データを `events` に移行する場合:

```sql
-- programs → events データ移行
INSERT INTO events (
    platform_id,
    source_id,
    external_event_id,
    type,
    title,
    description,
    start_at,
    end_at,
    published_at,
    url,
    image_url,
    created_at,
    updated_at
)
SELECT
    'youtube' as platform_id,  -- platform_nameから推測
    NULL as source_id,          -- 後で更新が必要
    id::text as external_event_id,  -- 仮のID
    'video' as type,            -- デフォルトはvideo
    title,
    NULL as description,
    start_at,
    end_at,
    start_at as published_at,   -- start_atをpublished_atとして使用
    link_url as url,
    image_url,
    created_at,
    now() as updated_at
FROM programs
WHERE platform_name = 'YouTube';

-- 注意: source_idは手動で設定する必要があります
```

### ステップ 4: sqlc コード再生成

```bash
cd backend
sqlc generate
```

### ステップ 5: アプリケーションコード更新

#### handlers/subscription.go の変更点

```go
// 旧: UpsertSourceParams
db.UpsertSourceParams{
    PlatformID:        platform,
    ExternalID:        details.ChannelID,
    Handle:            pgtype.Text{String: details.Handle, Valid: details.Handle != ""},
    DisplayName:       pgtype.Text{String: details.DisplayName, Valid: details.DisplayName != ""},
    ThumbnailUrl:      pgtype.Text{String: details.ThumbnailURL, Valid: details.ThumbnailURL != ""},
    UploadsPlaylistID: pgtype.Text{String: details.UploadsPlaylistID, Valid: details.UploadsPlaylistID != ""},
}

// 新: 同じ（変更なし）
```

```go
// 旧: UpsertUserSubscriptionParams
db.UpsertUserSubscriptionParams{
    UserID:   userID,  // int32
    SourceID: source.ID,
    Enabled:  true,
}

// 新: priorityパラメータ追加
db.UpsertUserSubscriptionParams{
    UserID:   int64(userID),  // int64に変更
    SourceID: source.ID,
    Enabled:  true,
    Priority: 0,  // 追加
}
```

#### main.go の変更点

```go
// 旧: ListPrograms
programs, err := queries.ListPrograms(ctx)

// 新: ListTimeline（ユーザーIDが必要）
timeline, err := queries.ListTimeline(ctx, db.ListTimelineParams{
    UserID:  1,  // ユーザーID
    Column2: pgtype.Timestamptz{Valid: false},  // before_time
    Limit:   50,
})
```

### ステップ 6: 旧テーブル削除（オプション）

移行が完了し、動作確認できたら旧テーブルを削除:

```sql
-- 注意: バックアップを取ってから実行
DROP TABLE IF EXISTS programs;
```

## コード更新チェックリスト

- [ ] `backend/sql/migrations/` 実行
- [ ] `sqlc generate` 実行
- [ ] `handlers/subscription.go` 更新
  - [ ] `UpsertUserSubscriptionParams` に `Priority` 追加
  - [ ] `UserID` の型を `int64` に変更
- [ ] `cmd/server/main.go` 更新
  - [ ] `ListPrograms` → `ListTimeline` に変更
  - [ ] タイムライン取得ロジックを更新
- [ ] YouTube 取り込みロジック更新
  - [ ] `programs` への保存 → `events` への保存
  - [ ] `UpsertEvent` を使用
- [ ] テスト実行
  - [ ] `go test ./...`
- [ ] 動作確認
  - [ ] 購読登録 API
  - [ ] タイムライン取得 API

## トラブルシューティング

### エラー: `column "priority" does not exist`

**原因**: `user_subscriptions` テーブルが古いスキーマのまま

**解決策**:

```bash
# テーブルを削除して再作成
psql $DATABASE_URL -c "DROP TABLE user_subscriptions CASCADE;"
psql $DATABASE_URL < sql/migrations/001_create_tables.sql
```

### エラー: `type mismatch: expected int64, got int32`

**原因**: `user_id` の型が変更された

**解決策**:

```go
// int32 → int64 にキャスト
UserID: int64(userID),
```

### エラー: `relation "programs" does not exist`

**原因**: 旧テーブル `programs` を参照している

**解決策**:

```go
// ListPrograms → ListTimeline に変更
timeline, err := queries.ListTimeline(ctx, db.ListTimelineParams{
    UserID:  userID,
    Column2: pgtype.Timestamptz{Valid: false},
    Limit:   50,
})
```

## ロールバック手順

問題が発生した場合のロールバック:

```bash
# 1. バックアップから復元
psql $DATABASE_URL < backup_YYYYMMDD.sql

# 2. 旧コードにrevert
git revert <commit-hash>

# 3. sqlc再生成
cd backend
sqlc generate
```

## 参考リンク

- [新スキーマ詳細](sql/README.md)
- [マイグレーションファイル](sql/migrations/)
- [sqlc クエリ](sql/queries/)
