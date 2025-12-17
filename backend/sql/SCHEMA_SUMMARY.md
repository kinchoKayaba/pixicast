# Pixicast スキーマ概要

## 📊 ER 図（テキスト版）

```
┌─────────────┐
│  platforms  │
│─────────────│
│ id (PK)     │◄─────┐
│ name        │      │
│ created_at  │      │
└─────────────┘      │
                     │
┌─────────────────────┼──────────────────────┐
│                     │                      │
│  ┌──────────────────▼────┐    ┌───────────▼──────┐
│  │      sources          │    │      events       │
│  │───────────────────────│    │──────────────────│
│  │ id (PK)               │◄───│ id (PK)          │
│  │ platform_id (FK)      │    │ platform_id (FK) │
│  │ external_id           │    │ source_id (FK)   │
│  │ handle                │    │ external_event_id│
│  │ display_name          │    │ type             │
│  │ thumbnail_url         │    │ title            │
│  │ uploads_playlist_id   │    │ description      │
│  │ last_fetched_at       │    │ start_at         │
│  │ fetch_status          │    │ end_at           │
│  │ created_at            │    │ published_at     │
│  │ updated_at            │    │ url              │
│  │ UNIQUE(platform_id,   │    │ image_url        │
│  │        external_id)   │    │ metrics (JSONB)  │
│  └───────────┬───────────┘    │ created_at       │
│              │                │ updated_at       │
│              │                │ UNIQUE(platform_id,
│              │                │   external_event_id)
│              │                └──────────────────┘
│              │
│  ┌───────────▼──────────────┐
│  │  user_subscriptions      │
│  │──────────────────────────│
│  │ user_id (PK)             │
│  │ source_id (PK, FK)       │
│  │ enabled                  │
│  │ priority                 │
│  │ created_at               │
│  │ updated_at               │
│  └──────────────────────────┘
```

## 🎯 設計思想

### 1. 正規化

- **platforms**: プラットフォームをマスタテーブルとして分離
- **sources**: チャンネル/配信者を一元管理
- **events**: タイムライン項目を正規化（旧 programs を置き換え）
- **user_subscriptions**: ユーザーと配信元の多対多関係

### 2. 拡張性

- **JSONB metrics**: 統計情報を柔軟に保存
- **fetch_status**: 取り込み状態を管理（エラーハンドリング）
- **priority**: 表示順序のカスタマイズ
- **type**: イベントタイプで分類（live/scheduled/video/premiere）

### 3. パフォーマンス

- **複合インデックス**: タイムライン取得を高速化
- **部分インデックス**: enabled=true のみインデックス化
- **COALESCE**: start_at と published_at を統一的に扱う

## 📝 主要クエリパターン

### 購読登録フロー

```sql
-- 1. チャンネル情報をupsert
INSERT INTO sources (...) VALUES (...)
ON CONFLICT (platform_id, external_id) DO UPDATE ...

-- 2. 購読情報をupsert
INSERT INTO user_subscriptions (...) VALUES (...)
ON CONFLICT (user_id, source_id) DO UPDATE ...
```

### タイムライン取得

```sql
SELECT e.*, s.display_name, s.thumbnail_url, s.handle
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE us.user_id = ? AND us.enabled = true
ORDER BY COALESCE(e.start_at, e.published_at) DESC
LIMIT ?;
```

### 配信中イベント取得

```sql
SELECT e.*, s.*
FROM events e
JOIN sources s ON e.source_id = s.id
JOIN user_subscriptions us ON s.id = us.source_id
WHERE
    us.user_id = ?
    AND us.enabled = true
    AND e.type = 'live'
    AND e.start_at <= now()
    AND (e.end_at IS NULL OR e.end_at > now())
ORDER BY e.start_at DESC;
```

## 🔑 制約とインデックス

### UNIQUE 制約

- `sources(platform_id, external_id)`: 同じチャンネルの重複登録を防止
- `events(platform_id, external_event_id)`: 同じイベントの重複登録を防止
- `user_subscriptions(user_id, source_id)`: 同じ購読の重複を防止

### インデックス

| テーブル           | インデックス                     | 用途                   |
| ------------------ | -------------------------------- | ---------------------- |
| sources            | idx_sources_platform_id          | プラットフォーム別検索 |
| sources            | idx_sources_fetch_status         | エラー状態の検索       |
| user_subscriptions | idx_user_subscriptions_source_id | 逆引き                 |
| user_subscriptions | idx_user_subscriptions_enabled   | 有効購読フィルタ       |
| events             | idx_events_source_published      | ソース別タイムライン   |
| events             | idx_events_start_at              | 開始時刻順ソート       |
| events             | idx_events_timeline              | タイムライン取得       |
| events             | idx_events_type                  | タイプ別検索           |

## 📈 データフロー

### 購読登録時

```
1. YouTube API → チャンネル情報取得
2. sources テーブルに upsert
3. user_subscriptions テーブルに upsert
4. 非同期でイベント取り込み開始
```

### イベント取り込み時

```
1. YouTube API → 動画/配信一覧取得
2. events テーブルに upsert
3. sources.last_fetched_at 更新
4. エラー時は fetch_status 更新
```

### タイムライン表示時

```
1. user_subscriptions で購読中のsource_id取得
2. events を JOIN して取得
3. COALESCE(start_at, published_at) でソート
4. ページネーション（before_time, limit）
```

## 🚀 パフォーマンス最適化

### クエリ最適化

- **JOIN 順序**: user_subscriptions → sources → events
- **WHERE 句**: enabled=true を先にフィルタ
- **インデックス活用**: COALESCE 用の複合インデックス

### データ削除戦略

```sql
-- 90日以上前の動画を削除（定期実行）
DELETE FROM events
WHERE type = 'video' AND published_at < now() - INTERVAL '90 days';
```

### 取り込み頻度制御

```sql
-- 10分以内に取り込み済みのソースは除外
SELECT * FROM sources
WHERE
    fetch_status = 'ok'
    AND (last_fetched_at IS NULL OR last_fetched_at < now() - INTERVAL '10 minutes')
ORDER BY last_fetched_at ASC NULLS FIRST;
```

## 🔧 運用 Tips

### 1. モニタリング

```sql
-- エラー状態のソース確認
SELECT * FROM sources WHERE fetch_status != 'ok';

-- イベント数の確認
SELECT
    s.display_name,
    COUNT(e.id) as event_count
FROM sources s
LEFT JOIN events e ON s.id = e.source_id
GROUP BY s.id, s.display_name
ORDER BY event_count DESC;

-- ユーザーごとの購読数
SELECT
    user_id,
    COUNT(*) as subscription_count
FROM user_subscriptions
WHERE enabled = true
GROUP BY user_id;
```

### 2. メンテナンス

```sql
-- 古いイベント削除
DELETE FROM events WHERE published_at < now() - INTERVAL '90 days';

-- 孤立したソース削除（購読者がいない）
DELETE FROM sources
WHERE id NOT IN (SELECT source_id FROM user_subscriptions);

-- 取り込みステータスリセット
UPDATE sources SET fetch_status = 'ok' WHERE fetch_status = 'error';
```

### 3. バックアップ

```bash
# PostgreSQL
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql

# CockroachDB
cockroach dump pixicast --url "$DATABASE_URL" > backup_$(date +%Y%m%d).sql
```

## 📚 関連ドキュメント

- [詳細スキーマ仕様](README.md)
- [マイグレーションガイド](../MIGRATION_GUIDE.md)
- [購読 API 仕様](../SUBSCRIPTION_API.md)
- [サンプルコード](../examples/timeline_example.go)

## 🎓 学習リソース

### PostgreSQL

- [JSONB 型](https://www.postgresql.org/docs/current/datatype-json.html)
- [部分インデックス](https://www.postgresql.org/docs/current/indexes-partial.html)
- [UPSERT (ON CONFLICT)](https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT)

### CockroachDB

- [PostgreSQL 互換性](https://www.cockroachlabs.com/docs/stable/postgresql-compatibility.html)
- [gen_random_uuid()](https://www.cockroachlabs.com/docs/stable/functions-and-operators.html#id-generation-functions)
- [JSONB](https://www.cockroachlabs.com/docs/stable/jsonb.html)
