# Pixicast

**「自分専用のコンテンツ編成表」**

YouTube、Twitch、ラジオなどの配信スケジュールを、ひとつのタイムラインで管理・可視化する Web アプリケーション。

## 🚀 コンセプト

「Google カレンダーに混ぜると予定が埋もれてしまう」「複数のプラットフォームを巡回するのが面倒」という課題を解決します。
自分が見たいコンテンツだけを集約し、テレビ欄のような感覚で「今、何がやっているか」を一目で把握できます。

## 🛠️ 技術スタック (Tech Stack)

モダンでスケーラブルな「Go × Next.js」構成を採用。gRPC (Connect) を用いた型安全な通信を実現しています。

| Category           | Technology                                            |
| :----------------- | :---------------------------------------------------- |
| **Frontend**       | **Next.js 16 (App Router)**, TypeScript, Tailwind CSS |
| **Backend**        | **Go (1.23)**, ConnectRPC (gRPC), net/http            |
| **Database**       | **CockroachDB Serverless** (PostgreSQL compatible)    |
| **ORM / Query**    | **sqlc** (Type-safe SQL generator), pgx               |
| **Auth**           | **NextAuth.js v5** (Google OAuth)                     |
| **Infrastructure** | **Google Cloud Run** (Backend), **Vercel** (Frontend) |
| **Tools**          | **Buf** (Protobuf management), Docker                 |

## 🌟 主な機能

- **タイムライン表示:** 複数の配信プラットフォームのスケジュールを時系列で統合表示。
- **ライブ判定:** 現在放送中の番組をリアルタイムでハイライト。
- **Google ログイン:** NextAuth.js によるセキュアな認証と、ユーザーごとのアイコン表示。
- **クラウドデータベース:** CockroachDB へのデータ永続化。

## 💻 ローカル開発環境のセットアップ

### 前提条件

- Go 1.23+
- Node.js 20+
- Buf CLI
- sqlc

### 1. リポジトリのクローン

```bash
git clone https://github.com/kinchoKayaba/pixicast.git
cd pixicast
```

### 2. 環境変数の設定

**backend/.env** (DB 接続用)

```env
DATABASE_URL="postgresql://user:pass@host:port/defaultdb?sslmode=verify-full"
```

**frontend/.env.local** (認証用)

```env
AUTH_GOOGLE_ID="your-google-client-id"
AUTH_GOOGLE_SECRET="your-google-client-secret"
AUTH_SECRET="random-string"
BACKEND_URL="http://localhost:8080" # ローカル開発時
```

### 3. コード生成 (gRPC & SQL)

Proto ファイルや SQL スキーマを変更した場合は実行してください。

```bash
# gRPCコード生成
PATH=$PATH:$(pwd)/frontend/node_modules/.bin buf generate proto --template buf.gen.yaml

# SQLコード生成
cd backend && sqlc generate
```

### 4. 起動

**Backend (Go)**

```bash
cd backend
go run cmd/server/main.go
```

**Frontend (Next.js)**

```bash
cd frontend
npm install
npm run dev
```

ブラウザで http://localhost:3000 にアクセス。

## 📂 ディレクトリ構成

- `proto/`: gRPC スキーマ定義 (Single Source of Truth)
- `backend/`: Go API サーバー
- `frontend/`: Next.js アプリケーション
