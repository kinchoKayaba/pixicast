# 🎬 Pixicast

**「自分専用のコンテンツ編成表」**

YouTube、Twitch、ラジオなどの配信スケジュールを、ひとつのタイムラインで管理・可視化する Web アプリケーション。

## 🚀 コンセプト

「Google カレンダーに混ぜると予定が埋もれてしまう」「複数のプラットフォームを巡回するのが面倒」という課題を解決します。
自分が見たいコンテンツだけを集約し、テレビ欄のような感覚で「今、何がやっているか」を一目で把握できます。

## 🛠️ 技術スタック

| Category           | Technology                                            |
| :----------------- | :---------------------------------------------------- |
| **Frontend**       | **Next.js 16 (App Router)**, TypeScript, Tailwind CSS |
| **Backend**        | **Go (1.25)**, ConnectRPC (gRPC), net/http           |
| **Database**       | **PostgreSQL 16** (開発), **CockroachDB** (本番)     |
| **ORM / Query**    | **sqlc** (Type-safe SQL generator), pgx              |
| **Auth**           | **Firebase Authentication**                           |
| **Infrastructure** | **Google Cloud Run**, **Vercel**, **Docker/OrbStack** |

## 💻 クイックスタート

### 前提条件

**🐳 Docker開発環境（推奨）:**
- **macOS**: [OrbStack](https://orbstack.dev/) (軽量・高速) または Docker Desktop
- **その他OS**: Docker & Docker Compose

### 1. リポジトリのクローン

```bash
git clone https://github.com/kinchoKayaba/pixicast.git
cd pixicast
```

### 2. 環境変数の設定

```bash
cp .env.docker .env
# .envファイルを編集してAPIキーを設定
```

### 3. Docker環境を起動

```bash
make dev
```

### 4. ブラウザでアクセス

```
http://localhost:3000
```

---

## 📋 開発コマンド

### 🐳 Docker開発環境（推奨）

```bash
make dev              # Docker環境起動
make docker-down      # Docker停止
make docker-logs      # ログ表示
make docker-restart   # 再起動
make docker-build     # イメージ再ビルド
```

### 💻 ローカル開発環境

```bash
make dev-local        # ローカル環境で起動
make dev-backend      # バックエンドのみ
make dev-frontend     # フロントエンドのみ
```

詳細は `make help` を参照してください。

---

## 📂 ディレクトリ構成

- `proto/`: gRPC スキーマ定義
- `backend/`: Go API サーバー
- `frontend/`: Next.js アプリケーション
- `docker-compose.yml`: Docker構成
- `Makefile`: 開発コマンド
