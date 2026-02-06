# Pixicast - Software Design Document (SDD)

## 📋 Document Information
- **Project:** Pixicast
- **Version:** 1.0.0
- **Last Updated:** 2026-02-06
- **Status:** In Development

---

## 1. Overview

### 1.1 Product Vision
Pixicastは「自分専用のコンテンツ編成表」を提供するWebアプリケーションです。YouTube、Twitch、Podcast、ラジオ、アニメ、TV番組など、複数のプラットフォームにまたがるコンテンツをひとつのタイムラインに集約し、ユーザーが「今、何がやっているか」を一目で把握できるようにします。

### 1.2 Core Concept
「Googleカレンダーに混ぜると予定が埋もれてしまう」「複数のプラットフォームを巡回するのが面倒」という課題を解決し、テレビ欄のような感覚でコンテンツを一覧できる体験を提供します。

### 1.3 Target Users
- 複数のYouTuberやストリーマーを追いかけているユーザー
- Podcast、ラジオ、アニメなど多様なコンテンツを視聴するユーザー
- 配信スケジュールを効率的に管理したいユーザー

---

## 2. Goals and Non-Goals

### 2.1 Goals
- ✅ **マルチプラットフォーム対応**: YouTube, Twitch, Podcast, Radiko, アニメ情報, TV情報を統合
- ✅ **パーソナライズド・タイムライン**: ユーザーが登録したチャンネル/番組のみを表示
- ✅ **リアルタイム更新**: ライブ配信の開始/終了を検知して表示
- ✅ **段階的な認証体験**: 未ログインでも利用可能、ログインで機能拡張
- ✅ **効率的なデータ取得**: API制限を考慮したバッチ処理とキャッシング
- ✅ **マネタイゼーション**: プラン別の機能制限（無料/有料）

### 2.2 Non-Goals
- ❌ **動画プレイヤーの実装**: 外部サイトへのリンクで対応
- ❌ **コメント機能**: SNS的な機能は提供しない
- ❌ **ライブチャット**: 各プラットフォームのチャット機能を利用
- ❌ **動画のダウンロード**: 著作権の観点から提供しない

---

## 3. Technical Architecture

### 3.1 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (Vercel)                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Next.js 16 (App Router) + TypeScript + Tailwind CSS │   │
│  │ - Timeline UI                                        │   │
│  │ - Channel Management                                 │   │
│  │ - Authentication Flow (Firebase Auth)               │   │
│  │ - Landing Page                                       │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ↓ gRPC (ConnectRPC) / REST API
┌─────────────────────────────────────────────────────────────┐
│                    Backend (Cloud Run)                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Go 1.25                                               │  │
│  │ - gRPC Server (ConnectRPC)                            │  │
│  │ - REST API (net/http)                                 │  │
│  │ - Firebase Auth Middleware                            │  │
│  │ - Rate Limiting & Caching                             │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ↓                 ↓                 ↓
┌──────────────────┐ ┌──────────────┐ ┌──────────────────┐
│   Database       │ │  External    │ │  Batch Jobs      │
│   (CockroachDB)  │ │  APIs        │ │  (Cloud Run Jobs)│
│  - users         │ │ - YouTube    │ │ - fetch_videos   │
│  - sources       │ │ - Twitch     │ │ - update_live    │
│  - events        │ │ - Podcast    │ │ - cleanup_anon   │
│  - subscriptions │ │ - Radiko*    │ │ - fetch_radiko*  │
│  - plan_limits   │ │ - Anime DB*  │ │ - fetch_anime*   │
└──────────────────┘ └──────────────┘ └──────────────────┘

* = 未実装
```

### 3.2 Technology Stack

| Category | Technology | Purpose |
|----------|-----------|---------|
| **Frontend** | Next.js 16 (App Router) | React framework, SSR/SSG |
| | TypeScript | Type safety |
| | Tailwind CSS | Styling |
| | ConnectRPC Client | gRPC communication |
| **Backend** | Go 1.25 | High-performance API server |
| | ConnectRPC (gRPC) | Efficient RPC communication |
| | net/http | REST API endpoints |
| | sqlc | Type-safe SQL query generation |
| | pgx/v5 | PostgreSQL driver |
| **Database** | CockroachDB (Production) | Distributed SQL, global scalability |
| | PostgreSQL 16 (Development) | Local development |
| **Authentication** | Firebase Authentication | User authentication & authorization |
| **Infrastructure** | Google Cloud Run | Serverless container deployment |
| | Vercel | Frontend hosting |
| | Docker / OrbStack | Local development |
| **External APIs** | YouTube Data API v3 | Fetch video/channel data |
| | Twitch Helix API | Fetch stream/user data |
| | RSS Feeds | Podcast episode data |
| | iTunes Search API | Apple Podcasts metadata |

---

## 4. Data Models

### 4.1 Entity Relationship Diagram

```
┌─────────────┐          ┌──────────────┐          ┌─────────────┐
│   users     │          │ plan_limits  │          │ platforms   │
├─────────────┤          ├──────────────┤          ├─────────────┤
│ id (PK)     │    ┌────→│ plan_type(PK)│          │ id (PK)     │
│ firebase_uid│    │     │ max_channels │          │ name        │
│ plan_type   │────┘     │ display_name │          │ created_at  │
│ email       │          │ price_monthly│          └─────────────┘
│ display_name│          │ has_favorites│                 │
│ is_anonymous│          │ has_device...│                 │
│ ...         │          └──────────────┘                 │
└─────────────┘                                           │
       │                                                  │
       │                                                  ↓
       │                                          ┌──────────────┐
       │                                          │   sources    │
       │                                          ├──────────────┤
       │         ┌────────────────────────────────│ id (PK)      │
       │         │                                │ platform_id  │
       │         │                                │ external_id  │
       ↓         ↓                                │ display_name │
┌──────────────────────┐                         │ handle       │
│ user_subscriptions   │                         │ thumbnail_url│
├──────────────────────┤                         │ fetch_status │
│ user_id (PK, FK)     │                         │ ...          │
│ source_id (PK, FK)   │                         └──────────────┘
│ enabled              │                                │
│ priority             │                                │
│ is_favorite          │                                │
│ last_accessed_at     │                                │
│ ...                  │                                │
└──────────────────────┘                                ↓
                                                ┌──────────────┐
                                                │   events     │
                                                ├──────────────┤
                                                │ id (PK)      │
                                                │ platform_id  │
                                                │ source_id(FK)│
                                                │ external_eve │
                                                │ type         │
                                                │ title        │
                                                │ start_at     │
                                                │ published_at │
                                                │ url          │
                                                │ image_url    │
                                                │ metrics      │
                                                │ duration     │
                                                └──────────────┘
```

### 4.2 Table Definitions

#### 4.2.1 users
ユーザー情報を管理するテーブル。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | BIGSERIAL | PRIMARY KEY | 内部ユーザーID |
| firebase_uid | TEXT | UNIQUE NOT NULL | Firebase UID |
| plan_type | TEXT | NOT NULL, DEFAULT 'free_anonymous' | プラン種別 |
| email | TEXT | NULLABLE | メールアドレス |
| display_name | TEXT | NULLABLE | 表示名 |
| photo_url | TEXT | NULLABLE | プロフィール画像URL |
| is_anonymous | BOOLEAN | NOT NULL, DEFAULT true | 匿名ユーザーフラグ |
| last_accessed_at | TIMESTAMPTZ | NOT NULL | 最終アクセス日時 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

**Indexes:**
- `idx_users_firebase_uid` on `firebase_uid`
- `idx_users_plan_type` on `plan_type`
- `idx_users_last_accessed_at` on `last_accessed_at`

#### 4.2.2 plan_limits
プラン別の機能制限を定義するテーブル。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| plan_type | TEXT | PRIMARY KEY | プラン種別 |
| max_channels | INT | NOT NULL | 最大登録チャンネル数 |
| display_name | TEXT | NOT NULL | プラン表示名 |
| price_monthly | INT | NULLABLE | 月額料金（円） |
| has_favorites | BOOLEAN | NOT NULL, DEFAULT false | お気に入り機能 |
| has_device_sync | BOOLEAN | NOT NULL, DEFAULT false | デバイス間同期 |
| description | TEXT | NULLABLE | プラン説明 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |

**Predefined Plans:**
- `free_anonymous`: 5チャンネル、30日でデータ削除、広告あり
- `free_login`: 無制限チャンネル、データ永久保存、広告あり
- `pro`: 無制限チャンネル、広告なし（月額500円）

#### 4.2.3 platforms
配信プラットフォームを定義するマスタテーブル。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | プラットフォームID (例: "youtube") |
| name | TEXT | NOT NULL | プラットフォーム名 (例: "YouTube") |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |

**Predefined Platforms:**
- `youtube`: YouTube
- `twitch`: Twitch
- `podcast`: Podcast
- `radiko`: Radiko（未実装）
- `anime`: アニメ（未実装）
- `tv`: TV番組（未実装）

#### 4.2.4 sources
チャンネル/配信者/番組の情報を管理するテーブル。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | 内部ソースID |
| platform_id | TEXT | NOT NULL, FK(platforms.id) | プラットフォームID |
| external_id | TEXT | NOT NULL | 外部ID (YouTubeのchannelId等) |
| handle | TEXT | NULLABLE | ハンドル名 (例: @username) |
| display_name | TEXT | NULLABLE | 表示名 |
| thumbnail_url | TEXT | NULLABLE | サムネイルURL |
| uploads_playlist_id | TEXT | NULLABLE | YouTube用アップロードプレイリストID |
| apple_podcast_url | TEXT | NULLABLE | Apple Podcasts URL |
| last_fetched_at | TIMESTAMPTZ | NULLABLE | 最終取得日時 |
| fetch_status | TEXT | NOT NULL, DEFAULT 'ok' | 取得ステータス |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

**Constraints:**
- UNIQUE(`platform_id`, `external_id`)

**Indexes:**
- `idx_sources_platform_id` on `platform_id`
- `idx_sources_fetch_status` on `fetch_status` WHERE `fetch_status != 'ok'`

**fetch_status values:**
- `ok`: 正常に取得可能
- `not_found`: チャンネルが削除/非公開
- `suspended`: BAN状態
- `error`: 取得エラー

#### 4.2.5 user_subscriptions
ユーザーの購読情報を管理するテーブル。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| user_id | BIGINT | NOT NULL, FK(users.id) | ユーザーID |
| source_id | UUID | NOT NULL, FK(sources.id) ON DELETE CASCADE | ソースID |
| enabled | BOOLEAN | NOT NULL, DEFAULT true | 有効フラグ |
| priority | INT | NOT NULL, DEFAULT 0 | 表示優先度 |
| is_favorite | BOOLEAN | NOT NULL, DEFAULT false | お気に入りフラグ |
| last_accessed_at | TIMESTAMPTZ | NOT NULL | 最終アクセス日時 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

**Constraints:**
- PRIMARY KEY(`user_id`, `source_id`)

**Indexes:**
- `idx_user_subscriptions_source_id` on `source_id`
- `idx_user_subscriptions_enabled` on (`user_id`, `enabled`) WHERE `enabled = true`
- `idx_user_subscriptions_last_accessed` on `last_accessed_at`

#### 4.2.6 events
タイムライン項目（動画/配信/予定等）を管理するテーブル。

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | 内部イベントID |
| platform_id | TEXT | NOT NULL, FK(platforms.id) | プラットフォームID |
| source_id | UUID | NOT NULL, FK(sources.id) ON DELETE CASCADE | ソースID |
| external_event_id | TEXT | NOT NULL | 外部イベントID (YouTubeのvideoId等) |
| type | TEXT | NOT NULL | イベントタイプ |
| title | TEXT | NOT NULL | タイトル |
| description | TEXT | NULLABLE | 説明文 |
| start_at | TIMESTAMPTZ | NULLABLE | 配信開始時刻 |
| end_at | TIMESTAMPTZ | NULLABLE | 配信終了時刻 |
| published_at | TIMESTAMPTZ | NULLABLE | 公開日時 |
| url | TEXT | NOT NULL | コンテンツURL |
| image_url | TEXT | NULLABLE | サムネイルURL |
| metrics | JSONB | NULLABLE | 統計情報 (JSON) |
| duration | TEXT | NULLABLE | 動画長 (HH:MM:SS) |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

**Constraints:**
- UNIQUE(`platform_id`, `external_event_id`)

**Indexes:**
- `idx_events_source_published` on (`source_id`, `published_at DESC NULLS LAST`)
- `idx_events_start_at` on (`start_at DESC NULLS LAST`)
- `idx_events_timeline` on (`source_id`, `COALESCE(start_at, published_at) DESC NULLS LAST`)
- `idx_events_type` on (`type`, `start_at DESC NULLS LAST`)

**type values:**
- `live`: ライブ配信中
- `scheduled`: 配信予定
- `video`: アーカイブ動画
- `premiere`: プレミア公開

**metrics format (JSON):**
```json
{
  "views": 12345,
  "likes": 678,
  "comments": 90
}
```

---

## 5. API Specifications

### 5.1 Authentication

#### 5.1.1 Firebase Authentication
- **Provider**: Firebase Authentication
- **Supported Methods**:
  - Google OAuth (ログインユーザー向け)
  - Anonymous Auth (未ログインユーザー向け)
- **Token Format**: Firebase ID Token (JWT)
- **Header**: `Authorization: Bearer <ID_TOKEN>`

#### 5.1.2 User Plan Management
プランはFirebase Custom Claimsで管理：
```json
{
  "plan_type": "free_anonymous",
  "user_id": 12345
}
```

### 5.2 REST API Endpoints

#### 5.2.1 POST /v1/subscriptions
チャンネル/番組を購読登録する。

**Request:**
```json
{
  "platform": "youtube",
  "input": "https://www.youtube.com/@channel または UCxxx... または @handle"
}
```

**Response (201 Created):**
```json
{
  "subscription": {
    "user_id": 12345,
    "platform": "youtube",
    "source_id": "uuid",
    "channel_id": "UCxxx...",
    "handle": "channel",
    "display_name": "Channel Name",
    "thumbnail_url": "https://...",
    "enabled": true,
    "is_favorite": false
  }
}
```

**Error Responses:**
- `400`: Invalid input format
- `401`: Authentication required
- `403`: Channel limit reached for current plan
- `404`: Channel not found

#### 5.2.2 GET /v1/subscriptions
ユーザーの購読チャンネル一覧を取得する。

**Response (200 OK):**
```json
{
  "subscriptions": [
    {
      "user_id": 12345,
      "platform": "youtube",
      "source_id": "uuid",
      "channel_id": "UCxxx...",
      "handle": "channel",
      "display_name": "Channel Name",
      "thumbnail_url": "https://...",
      "enabled": true,
      "is_favorite": false
    }
  ]
}
```

#### 5.2.3 DELETE /v1/subscriptions/{channelId}
チャンネル登録を解除する。

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Subscription deleted successfully"
}
```

#### 5.2.4 POST /v1/subscriptions/{channelId}/favorite
お気に入り状態を切り替える（Basic以上のプラン限定）。

**Request:**
```json
{
  "is_favorite": true
}
```

**Response (200 OK):**
```json
{
  "success": true
}
```

**Error Response:**
- `403`: お気に入り機能はGoogleログイン後に利用できます

#### 5.2.5 GET /v1/me
ユーザー情報とプラン情報を取得する。

**Response (200 OK):**
```json
{
  "user": {
    "id": 12345,
    "firebase_uid": "xxx",
    "plan_type": "free_login",
    "email": "user@example.com",
    "display_name": "User Name",
    "photo_url": "https://...",
    "is_anonymous": false
  },
  "plan": {
    "type": "free_login",
    "display_name": "ベーシックプラン",
    "max_channels": 999999,
    "price_monthly": null,
    "has_favorites": true,
    "has_device_sync": true,
    "description": "ログインユーザー向け..."
  },
  "current_channels": 10
}
```

### 5.3 gRPC API Endpoints (ConnectRPC)

#### 5.3.1 GetTimeline
タイムラインを取得する。

**Request (timeline.proto):**
```protobuf
message GetTimelineRequest {
  string date = 1;                           // 日付 (YYYY-MM-DD)
  repeated string youtube_channel_ids = 2;   // YouTubeチャンネルIDリスト
  string before_time = 3;                    // カーソル（ページネーション用）
  int32 limit = 4;                           // 取得件数
}
```

**Response:**
```protobuf
message GetTimelineResponse {
  repeated Program programs = 1;
  bool has_more = 2;
  string next_cursor = 3;
}

message Program {
  string id = 1;
  string platform_name = 2;
  string channel_id = 3;
  string channel_title = 4;
  string channel_thumbnail_url = 5;
  string title = 6;
  string start_at = 7;
  string published_at = 8;
  string link_url = 9;
  string image_url = 10;
  bool is_live = 11;
  string duration = 12;
  int64 view_count = 13;
}
```

---

## 6. Business Logic

### 6.1 User Registration & Authentication Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. First Visit (未ログイン)                                  │
│    - Firebase Anonymous Authで自動ログイン                   │
│    - plan_type: "free_anonymous"                            │
│    - 最大5チャンネル登録可能                                  │
│    - 最終アクセスから30日でデータ削除                         │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. Google Login (ログイン)                                   │
│    - Firebase Google Authでログイン                          │
│    - Anonymous → Google へアカウントリンク                   │
│    - plan_type: "free_login" へアップグレード                │
│    - 無制限チャンネル登録可能                                 │
│    - お気に入り機能、デバイス間同期が利用可能                 │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Pro Plan (有料プラン) ※未実装                             │
│    - サブスクリプション決済（Stripe等）                       │
│    - plan_type: "pro"                                       │
│    - 広告非表示                                               │
│    - 全機能利用可能                                           │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 Channel Registration Flow

```
1. ユーザーがチャンネルURLを入力
   ↓
2. Backend: プラン制限チェック
   - 現在の登録チャンネル数をカウント
   - plan_limits.max_channelsと比較
   - 超過していれば403エラー
   ↓
3. Backend: チャンネル情報を取得
   - YouTube: ResolveHandle → GetChannelDetails
   - Twitch: GetUserByLogin
   - Podcast: ResolveFeedURL → ParseFeed
   ↓
4. Backend: DBに保存
   - sources テーブルにUPSERT
   - user_subscriptions テーブルにUPSERT
   ↓
5. Backend: 非同期で過去動画を取得（goroutine）
   - 2025/1/1以降の全動画を取得
   - events テーブルに保存
   ↓
6. Response: 201 Created
```

### 6.3 Timeline Generation Flow

```
1. Frontend: ユーザーがタイムラインをリクエスト
   ↓
2. Backend: 購読チャンネル一覧を取得
   - user_subscriptions からユーザーの購読チャンネルを取得
   ↓
3. Backend: イベント一覧を取得
   - events テーブルから該当チャンネルのイベントを取得
   - ライブ配信を最優先でソート
   - 最新順にソート
   ↓
4. Backend: レスポンスを返す
   - has_more: 続きがあるかどうか
   - next_cursor: 次のページのカーソル
   ↓
5. Frontend: タイムラインを表示
   - 日付でグループ化
   - 無限スクロールで追加読み込み
```

---

## 7. Batch Processing & Data Ingestion

### 7.1 Batch Job Architecture

```
┌────────────────────────────────────────────────────────────┐
│           Cloud Scheduler (Cron Jobs)                       │
│  - fetch_videos:     毎時00分                               │
│  - update_live:      5分ごと                                │
│  - cleanup_anon:     毎日04:00                              │
│  - fetch_radiko:     毎日06:00 (未実装)                     │
│  - fetch_anime:      毎日07:00 (未実装)                     │
└────────────────────────────────────────────────────────────┘
                           ↓
┌────────────────────────────────────────────────────────────┐
│               Cloud Run Jobs                                │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ fetch_videos                                          │ │
│  │ - 全チャンネルの新着動画を取得                        │ │
│  │ - YouTube API制限を考慮して段階的に実行              │ │
│  │ - 人気チャンネルは優先的に更新                        │ │
│  └──────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ update_live_status                                    │ │
│  │ - ライブ配信中/予定のステータスを更新                 │ │
│  │ - YouTube/Twitchのライブ配信を監視                   │ │
│  └──────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────┐ │
│  │ cleanup_anonymous                                     │ │
│  │ - 30日間アクセスのない匿名ユーザーを削除              │ │
│  └──────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────┘
```

### 7.2 YouTube API Quota Management

#### 7.2.1 API制限
- **1日あたりの上限**: 10,000 units
- **主なコスト**:
  - `channels.list`: 1 unit
  - `videos.list`: 1 unit
  - `playlistItems.list`: 1 unit
  - `search.list`: 100 units

#### 7.2.2 最適化戦略
1. **チャンネル優先度**
   - ユーザー登録数でチャンネルに優先度を付与
   - 人気チャンネルを優先的に更新

2. **更新頻度の調整**
   - **高優先度** (利用者50%以上): 1時間ごと
   - **中優先度** (利用者10-50%): 3時間ごと
   - **低優先度** (利用者10%未満): 6時間ごと

3. **キャッシング戦略**
   - Redis/Memcachedでチャンネル情報をキャッシュ
   - TTL: 高優先度=1h, 中優先度=3h, 低優先度=6h

4. **スマートスケジューリング**
   - 配信者の更新パターンを学習
     - 例: 毎週金曜18時 → 金曜17:45に更新
     - 例: 月水金18時 → 該当曜日の17:45に更新
   - 更新パターンをDBに保存（未実装）

#### 7.2.3 Batch実行フロー

```sql
-- チャンネル優先度の計算（例）
WITH channel_popularity AS (
  SELECT
    source_id,
    COUNT(DISTINCT user_id) as subscriber_count,
    MAX(last_accessed_at) as last_access
  FROM user_subscriptions
  WHERE enabled = true
  GROUP BY source_id
),
total_users AS (
  SELECT COUNT(DISTINCT user_id) as total FROM user_subscriptions
)
SELECT
  s.id,
  s.external_id,
  cp.subscriber_count,
  (cp.subscriber_count::float / tu.total) as popularity_ratio,
  CASE
    WHEN (cp.subscriber_count::float / tu.total) >= 0.5 THEN 'high'
    WHEN (cp.subscriber_count::float / tu.total) >= 0.1 THEN 'medium'
    ELSE 'low'
  END as priority,
  s.last_fetched_at
FROM sources s
JOIN channel_popularity cp ON s.id = cp.source_id
CROSS JOIN total_users tu
WHERE s.platform_id = 'youtube'
  AND s.fetch_status = 'ok'
ORDER BY priority DESC, s.last_fetched_at ASC NULLS FIRST;
```

### 7.3 Caching Strategy

```
┌──────────────────────────────────────────────────────────┐
│               Cache Layers                                │
│                                                           │
│  ┌────────────────────────────────────────────────────┐ │
│  │ Level 1: Application Memory (Go Map)                │ │
│  │ - 超高頻度アクセス（上位1%）                        │ │
│  │ - TTL: 5分                                          │ │
│  │ - 例: ヒカキン、はじめしゃちょー等                  │ │
│  └────────────────────────────────────────────────────┘ │
│                      ↓ (Cache Miss)                      │
│  ┌────────────────────────────────────────────────────┐ │
│  │ Level 2: Redis/Memcached (未実装)                   │ │
│  │ - 高頻度アクセス（上位10%）                         │ │
│  │ - TTL: 1-6時間（優先度により変動）                 │ │
│  └────────────────────────────────────────────────────┘ │
│                      ↓ (Cache Miss)                      │
│  ┌────────────────────────────────────────────────────┐ │
│  │ Level 3: Database (CockroachDB)                     │ │
│  │ - 全データ                                          │ │
│  │ - Index最適化でクエリ高速化                         │ │
│  └────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

---

## 8. Feature Roadmap

### 8.1 Phase 1: MVP (✅ 完了)
- [x] YouTube対応
- [x] Twitch対応
- [x] Podcast対応
- [x] Firebase Authentication (Anonymous + Google)
- [x] プラン別機能制限
- [x] タイムライン表示
- [x] チャンネル登録/解除
- [x] お気に入り機能

### 8.2 Phase 2: LP & UX改善（進行中）
- [ ] ランディングページ作成
  - [ ] ヒーローセクション
  - [ ] 機能紹介
  - [ ] プラン比較表
  - [ ] FAQ
- [ ] オンボーディングフロー改善
  - [ ] チュートリアル
  - [ ] サンプルチャンネルのレコメンド
- [ ] レスポンシブデザイン最適化

### 8.3 Phase 3: 新プラットフォーム対応（未実装）
- [ ] Radiko対応
  - [ ] Radiko APIインテグレーション
  - [ ] ラジオ番組のタイムテーブル取得
  - [ ] エリア別対応
- [ ] アニメ情報対応
  - [ ] しょぼいカレンダー or AniList API
  - [ ] 放送スケジュール取得
- [ ] TV番組情報対応
  - [ ] EPG（電子番組表）API
  - [ ] 地上波/BS/CS対応

### 8.4 Phase 4: バッチ処理最適化（未実装）
- [ ] スマートスケジューリング
  - [ ] 配信者の更新パターン学習
  - [ ] 曜日/時刻別の最適化
- [ ] Redisキャッシング導入
  - [ ] チャンネル情報のキャッシュ
  - [ ] イベント情報のキャッシュ
- [ ] YouTube API Quota監視
  - [ ] リアルタイムQuota残量表示
  - [ ] アラート機能

### 8.5 Phase 5: マネタイゼーション（未実装）
- [ ] Proプラン決済機能
  - [ ] Stripe統合
  - [ ] サブスクリプション管理
- [ ] 広告配信
  - [ ] Google AdSense統合
  - [ ] Free/Basic プランのみ表示

### 8.6 Phase 6: コミュニティ機能（検討中）
- [ ] チャンネルレコメンデーション
  - [ ] 協調フィルタリング
  - [ ] 類似ユーザーの購読傾向
- [ ] 通知機能
  - [ ] ライブ配信開始通知
  - [ ] 新着動画通知
  - [ ] プッシュ通知（Web Push）

---

## 9. Security & Privacy

### 9.1 Authentication & Authorization
- Firebase ID Tokenによる認証
- Custom Claimsでプラン情報を管理
- API Gatewayレベルでのレート制限

### 9.2 Data Privacy
- 個人情報の最小化（メール、表示名、写真のみ）
- 匿名ユーザーは30日で自動削除
- ログインユーザーはいつでもアカウント削除可能

### 9.3 External API Security
- API Keyは環境変数で管理
- Google Secret Managerで機密情報を管理
- API Keyのローテーション

---

## 10. Monitoring & Observability

### 10.1 Logging
- **Application Logs**: Cloud Logging (旧Stackdriver)
- **Access Logs**: Cloud Run自動ログ
- **Error Tracking**: Sentry（未実装）

### 10.2 Metrics
- **Backend**: Prometheus + Grafana（未実装）
  - API レスポンスタイム
  - エラー率
  - リクエスト数
- **Frontend**: Google Analytics（未実装）
  - ページビュー
  - ユーザー行動

### 10.3 Alerting
- Cloud Monitoring Alerts
  - API エラー率 > 5%
  - レスポンスタイム > 3秒
  - YouTube API Quota残量 < 1000

---

## 11. Deployment & CI/CD

### 11.1 Environments
- **Development**: Docker (OrbStack) + PostgreSQL
- **Staging**: Cloud Run (未実装)
- **Production**: Cloud Run + CockroachDB

### 11.2 CI/CD Pipeline
- **Git Flow**: main ブランチのみ
- **GitHub Actions**: (未実装)
  - `go test` + `golangci-lint`
  - Docker build & push
  - Cloud Run deploy
  - Frontend deploy to Vercel

### 11.3 Rollback Strategy
- Cloud Runのリビジョン管理
- 問題発生時は即座に前バージョンへロールバック

---

## 12. Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| API Response Time (p95) | < 200ms | ~100ms |
| Timeline Load Time | < 1s | ~800ms |
| Batch Job Duration (fetch_videos) | < 5min | ~3min |
| Database Query Time (p95) | < 50ms | ~30ms |
| Uptime | 99.9% | - |

---

## 13. Open Questions & TODOs

### 13.1 Technical Decisions
- [ ] Redis/Memcached どちらを使うか？
- [ ] Batch JobのスケジューリングツールはCloud Schedulerで十分か？
- [ ] Radiko APIの利用可否を確認

### 13.2 Business Decisions
- [ ] Proプランの価格設定（月額500円は適切か？）
- [ ] 広告配信の実装時期
- [ ] チャンネル登録上限（無制限は本当に良いか？）

### 13.3 Implementation TODOs
- [ ] Redisキャッシング層の追加
- [ ] YouTube API Quota監視ダッシュボード
- [ ] ランディングページデザイン
- [ ] Radiko/アニメ/TV対応のスコープ確定
- [ ] スマートスケジューリングのアルゴリズム設計

---

## 14. Appendix

### 14.1 Reference Documents
- [YouTube Data API v3 Documentation](https://developers.google.com/youtube/v3)
- [Twitch Helix API Documentation](https://dev.twitch.tv/docs/api/)
- [Firebase Authentication Documentation](https://firebase.google.com/docs/auth)
- [ConnectRPC Documentation](https://connectrpc.com/)
- [CockroachDB Documentation](https://www.cockroachlabs.com/docs/)

### 14.2 Glossary
- **SDD**: Software Design Document（ソフトウェア設計書）
- **MVP**: Minimum Viable Product（最小実行可能製品）
- **TTL**: Time To Live（キャッシュの有効期限）
- **EPG**: Electronic Program Guide（電子番組表）
- **Quota**: API利用制限
- **ConnectRPC**: gRPCのHTTP/2プロトコルをHTTP/1.1でも使えるようにしたRPCフレームワーク

---

**Document Version History:**
- v1.0.0 (2026-02-06): Initial version
