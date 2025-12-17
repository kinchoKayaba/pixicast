# Pixicast Database Schema

## 概要

Pixicast の購読管理とタイムライン表示を支えるデータベーススキーマです。
PostgreSQL 12+ / CockroachDB 21+ 互換で設計されています。

## テーブル構成

### platforms

配信プラットフォーム（YouTube, Twitch 等）のマスタテーブル

| カラム     | 型          | 説明                                      |
| ---------- | ----------- | ----------------------------------------- |
| id         | TEXT PK     | プラットフォーム ID ('youtube', 'twitch') |
| name       | TEXT        | 表示名                                    |
| created_at | TIMESTAMPTZ | 作成日時                                  |

### sources

チャンネル/配信者の情報

| カラム              | 型          | 説明                                          |
| ------------------- | ----------- | --------------------------------------------- |
| id                  | UUID PK     | 内部 ID                                       |
| platform_id         | TEXT FK     | プラットフォーム ID                           |
| external_id         | TEXT        | 外部 ID (YouTube channelId 等)                |
| handle              | TEXT        | @handle (nullable)                            |
| display_name        | TEXT        | 表示名                                        |
| thumbnail_url       | TEXT        | サムネイル URL                                |
| uploads_playlist_id | TEXT        | アップロードプレイリスト ID (YouTube 用)      |
| last_fetched_at     | TIMESTAMPTZ | 最終取得日時                                  |
| fetch_status        | TEXT        | 取得ステータス (ok/not_found/suspended/error) |
| created_at          | TIMESTAMPTZ | 作成日時                                      |
| updated_at          | TIMESTAMPTZ | 更新日時                                      |

**制約**: UNIQUE(platform_id, external_id)

### user_subscriptions

ユーザーの購読情報

| カラム     | 型          | 説明                 |
| ---------- | ----------- | -------------------- |
| user_id    | BIGINT PK   | ユーザー ID          |
| source_id  | UUID PK FK  | ソース ID            |
| enabled    | BOOLEAN     | 有効/無効            |
| priority   | INT         | 優先度（表示順序用） |
| created_at | TIMESTAMPTZ | 購読日時             |
| updated_at | TIMESTAMPTZ | 更新日時             |

**制約**: PRIMARY KEY(user_id, source_id)

### events

タイムライン項目（動画/配信/予定等）

| カラム            | 型          | 説明                                           |
| ----------------- | ----------- | ---------------------------------------------- |
| id                | UUID PK     | 内部 ID                                        |
| platform_id       | TEXT FK     | プラットフォーム ID                            |
| source_id         | UUID FK     | ソース ID                                      |
| external_event_id | TEXT        | 外部イベント ID (YouTube videoId 等)           |
| type              | TEXT        | イベントタイプ (live/scheduled/video/premiere) |
| title             | TEXT        | タイトル                                       |
| description       | TEXT        | 説明 (nullable)                                |
| start_at          | TIMESTAMPTZ | 開始日時 (nullable)                            |
| end_at            | TIMESTAMPTZ | 終了日時 (nullable)                            |
| published_at      | TIMESTAMPTZ | 公開日時 (nullable)                            |
| url               | TEXT        | URL                                            |
| image_url         | TEXT        | サムネイル URL                                 |
| metrics           | JSONB       | 統計情報 (nullable)                            |
| created_at        | TIMESTAMPTZ | 作成日時                                       |
| updated_at        | TIMESTAMPTZ | 更新日時                                       |

**制約**: UNIQUE(platform_id, external_event_id)

## マイグレーション

### 初回セットアップ

```bash
# 1. データベース接続確認
psql $DATABASE_URL -c "SELECT version();"

# 2. テーブル作成
psql $DATABASE_URL < sql/migrations/001_create_tables.sql

# 3. 初期データ投入
psql $DATABASE_URL < sql/migrations/002_seed_platforms.sql

# 4. 確認
psql $DATABASE_URL -c "\dt"
psql $DATABASE_URL -c "SELECT * FROM platforms;"
```

### CockroachDB の場合

```bash
# CockroachDB Cloud の場合
cockroach sql --url "$DATABASE_URL" < sql/migrations/001_create_tables.sql
cockroach sql --url "$DATABASE_URL" < sql/migrations/002_seed_platforms.sql
```

### マイグレーションファイル

- `001_create_tables.sql`: 全テーブル作成
- `002_seed_platforms.sql`: プラットフォームマスタ初期データ

## sqlc クエリ

### query_sources.sql

ソース（チャンネル）管理用クエリ

- `UpsertSource`: チャンネル情報の upsert
- `GetSourceByID`: ID 検索
- `GetSourceByExternalID`: 外部 ID 検索
- `ListSources`: 一覧取得
- `ListSourcesByPlatform`: プラットフォーム別一覧
- `UpdateSourceFetchStatus`: 取得ステータス更新
- `ListSourcesForFetch`: 取り込み対象取得

### query_subscriptions.sql

購読管理用クエリ

- `UpsertUserSubscription`: 購読情報の upsert
- `GetUserSubscription`: 購読情報取得
- `ListUserSubscriptions`: ユーザーの購読一覧（全て）
- `ListUserEnabledSubscriptions`: ユーザーの有効な購読一覧
- `UpdateSubscriptionEnabled`: 有効/無効切り替え
- `UpdateSubscriptionPriority`: 優先度更新
- `DeleteUserSubscription`: 購読削除
- `CountUserSubscriptions`: 購読数カウント
- `ListSubscribedSourceIDs`: 購読中の source_id リスト

### query_timeline.sql

タイムライン管理用クエリ

- `UpsertEvent`: イベント情報の upsert
- `GetEventByID`: ID 検索
- `GetEventByExternalID`: 外部 ID 検索
- `ListTimeline`: ユーザーのタイムライン取得（ページネーション対応）
- `ListTimelineBySource`: ソース別タイムライン
- `ListLiveEvents`: 配信中イベント一覧
- `ListUpcomingEvents`: 今後予定されているイベント一覧
- `ListEventsByType`: タイプ別イベント一覧
- `DeleteOldEvents`: 古いイベント削除
- `CountEventsBySource`: ソース別イベント数

## sqlc コード生成

```bash
cd backend
sqlc generate
```

生成されるファイル:

- `db/models.go`: テーブル構造体
- `db/query_sources.sql.go`: ソース管理関数
- `db/query_subscriptions.sql.go`: 購読管理関数
- `db/query_timeline.sql.go`: タイムライン管理関数

## 使用例

### 購読登録フロー

```go
// 1. チャンネル情報をupsert
source, err := queries.UpsertSource(ctx, db.UpsertSourceParams{
    PlatformID:        "youtube",
    ExternalID:        "UCxxxxxxxxxxxx",
    Handle:            pgtype.Text{String: "junchannel", Valid: true},
    DisplayName:       pgtype.Text{String: "Jun Channel", Valid: true},
    ThumbnailUrl:      pgtype.Text{String: "https://...", Valid: true},
    UploadsPlaylistID: pgtype.Text{String: "UUxxxxxxxxxxxx", Valid: true},
})

// 2. ユーザー購読をupsert
subscription, err := queries.UpsertUserSubscription(ctx, db.UpsertUserSubscriptionParams{
    UserID:   1,
    SourceID: source.ID,
    Enabled:  true,
    Priority: 0,
})
```

### タイムライン取得

```go
// ユーザーのタイムラインを取得（最新50件）
timeline, err := queries.ListTimeline(ctx, db.ListTimelineParams{
    UserID: 1,
    Column2: pgtype.Timestamptz{Valid: false}, // before_time なし
    Limit: 50,
})

for _, item := range timeline {
    fmt.Printf("%s: %s by %s\n",
        item.Type,
        item.Title,
        item.SourceDisplayName.String)
}
```

### イベント登録

```go
// YouTube動画をイベントとして登録
event, err := queries.UpsertEvent(ctx, db.UpsertEventParams{
    PlatformID:      "youtube",
    SourceID:        sourceID,
    ExternalEventID: "dQw4w9WgXcQ",
    Type:            "video",
    Title:           "Sample Video",
    Description:     pgtype.Text{String: "Description", Valid: true},
    PublishedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
    Url:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    ImageUrl:        pgtype.Text{String: "https://...", Valid: true},
    Metrics:         []byte(`{"views": 1000, "likes": 50}`),
})
```

## インデックス戦略

### sources

- `idx_sources_platform_id`: プラットフォーム別検索
- `idx_sources_fetch_status`: エラー状態の検索

### user_subscriptions

- `idx_user_subscriptions_source_id`: 逆引き（ソース → 購読者）
- `idx_user_subscriptions_enabled`: 有効な購読のみ

### events

- `idx_events_source_published`: ソース別タイムライン
- `idx_events_start_at`: 開始時刻順ソート
- `idx_events_timeline`: タイムライン取得用複合インデックス
- `idx_events_type`: タイプ別検索

## パフォーマンス考慮事項

1. **タイムライン取得**: `COALESCE(start_at, published_at)` でソート

   - ライブ配信は `start_at` を使用
   - 動画は `published_at` を使用
   - 複合インデックスで高速化

2. **購読フィルタ**: `enabled = true` の部分インデックス

3. **古いイベント削除**: 90 日以上前の動画を定期削除

4. **JSONB metrics**: 統計情報は柔軟に拡張可能

## 今後の拡張

- [ ] users テーブル追加（認証実装時）
- [ ] notifications テーブル（通知機能）
- [ ] tags テーブル（タグ・カテゴリ機能）
- [ ] user_event_states テーブル（既読/お気に入り等）
- [ ] マイグレーションツール導入（golang-migrate 等）
