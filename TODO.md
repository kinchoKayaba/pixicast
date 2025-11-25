# 開発ロードマップ: コンテンツ配信タイムラインアプリ

## コンセプト

自分専用の視聴スケジュールアプリ。YouTube、ラジオ、アニメ、スポーツ等の配信予定を時系列（タイムライン）で可視化する。

## 技術スタック

- **Backend:** Go (Connect/gRPC), Cloud Run, CockroachDB
- **Frontend:** TypeScript, Next.js (App Router), Vercel
- **Auth:** NextAuth.js (Google OAuth)

---

## 📅 Phase 1: 開通 (Walking Skeleton)

機能は空っぽでも、技術スタック全体が繋がっている状態を作る。

- [ ] **プロジェクト作成**
  - [ ] Git リポジトリ作成 (Monorepo 構成: `/backend`, `/frontend`)
  - [ ] `go mod init`
  - [ ] Next.js プロジェクト作成 (`npx create-next-app`)
- [ ] **gRPC/Connect 環境構築**
  - [ ] `buf` のインストールと設定
  - [ ] `.proto` ファイル作成 (Hello World API 定義)
  - [ ] コード自動生成 (Go サーバーコード & TS クライアントコード)
- [ ] **疎通確認**
  - [ ] Go サーバー起動 (localhost)
  - [ ] Next.js から gRPC クライアントでリクエスト送信
  - [ ] ブラウザ画面にサーバーからのレスポンスが表示されることを確認

## 🎨 Phase 2: UI プロトタイプ (Mocking)

DB は使わず、固定データで理想の UI を作り込む。

- [ ] **API 定義の具体化**
  - [ ] `timeline.proto` に本番用のデータ構造を定義 (Title, StartTime, ImageURL, etc.)
  - [ ] 再生成 (`buf generate`)
- [ ] **モックサーバー実装**
  - [ ] Go 側で固定のダミーデータ (JSON/Struct) を返す実装をする
- [ ] **フロントエンド実装**
  - [ ] タイムライン表示コンポーネント作成
  - [ ] 現在時刻のインジケータ (赤線) 表示
  - [ ] スマホ表示対応 (レスポンシブ調整)

## 🗄️ Phase 3: DB 接続 (Database Integration)

実際のデータを保存・取得できるようにする。

- [ ] **DB 環境構築**
  - [ ] CockroachDB Serverless アカウント作成・クラスタ作成
  - [ ] 接続確認
- [ ] **データアクセス層の実装**
  - [ ] テーブル設計 (DDL 作成: `programs`, `channels` etc.)
  - [ ] マイグレーションツールの導入 (golang-migrate / atlas 等)
  - [ ] Go 側での DB 接続実装 (pgx / Gorm / Ent 等)
- [ ] **CRUD 実装**
  - [ ] データ取得処理をモックから DB クエリに差し替え
  - [ ] (簡易的で OK) データ登録用の API エンドポイント作成
  - [ ] curl や Postman で予定を追加し、ブラウザに反映されるか確認

## 🔒 Phase 4: 個人化 (Authentication)

自分専用のデータのみを扱うように制限をかける。

- [ ] **認証機能の実装**
  - [ ] NextAuth.js 導入 (Google Provider 設定)
  - [ ] ログイン画面・ログアウト処理
- [ ] **バックエンド連携**
  - [ ] リクエストヘッダーへのトークン付与
  - [ ] Go 側での JWT 検証ミドルウェア実装
- [ ] **ユーザー別データ管理**
  - [ ] DB テーブルに `user_id` カラムを追加
  - [ ] クエリに `WHERE user_id = ?` を追加

## 🚀 Phase 5: 本番化 (Deployment & CI/CD)

クラウド環境へデプロイし、自動化パイプラインを組む。

- [ ] **コンテナ化**
  - [ ] Backend 用の `Dockerfile` 作成・ビルド確認
- [ ] **デプロイ設定**
  - [ ] Google Cloud Run へのデプロイ
  - [ ] Vercel へのデプロイ
  - [ ] 環境変数 (DB URL, Auth Secret) の設定
- [ ] **CI/CD (GitHub Actions)**
  - [ ] `go test` & `golangci-lint` の自動実行
  - [ ] main ブランチへのプッシュで自動デプロイ
