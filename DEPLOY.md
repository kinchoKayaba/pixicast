# Pixicast デプロイガイド

## 🚀 デプロイ概要

- **バックエンド**: Google Cloud Run
- **フロントエンド**: Vercel
- **データベース**: CockroachDB (既存)

---

## 📋 事前準備

### 1. 必要なツール

- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install)
- [Vercel CLI](https://vercel.com/docs/cli) (オプション)
- Git

### 2. Google Cloudプロジェクト設定

```bash
# プロジェクトID
export GCP_PROJECT_ID="pixicast"

# gcloudの初期化
gcloud init
gcloud config set project $GCP_PROJECT_ID

# 必要なAPIを有効化
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com
```

### 3. Artifact Registryリポジトリ作成

```bash
gcloud artifacts repositories create pixicast \
  --repository-format=docker \
  --location=asia-northeast1 \
  --description="Pixicast Docker repository"
```

---

## 🔐 環境変数とシークレットの設定

### Google Cloud Secret Manager

```bash
# DATABASE_URL
echo -n "YOUR_DATABASE_URL" | \
  gcloud secrets create DATABASE_URL --data-file=-

# YOUTUBE_API_KEY
echo -n "YOUR_YOUTUBE_API_KEY" | \
  gcloud secrets create YOUTUBE_API_KEY --data-file=-

# TWITCH_CLIENT_ID
echo -n "YOUR_TWITCH_CLIENT_ID" | \
  gcloud secrets create TWITCH_CLIENT_ID --data-file=-

# TWITCH_CLIENT_SECRET
echo -n "YOUR_TWITCH_CLIENT_SECRET" | \
  gcloud secrets create TWITCH_CLIENT_SECRET --data-file=-

# Firebase Admin SDK (JSONファイル)
gcloud secrets create FIREBASE_ADMIN_KEY \
  --data-file=backend/pixicast-firebase-adminsdk-fbsvc-8e0eba3cbe.json
```

### Cloud Runサービスアカウントに権限付与

```bash
# デフォルトのCompute Engine サービスアカウント
PROJECT_NUMBER=$(gcloud projects describe $GCP_PROJECT_ID --format="value(projectNumber)")
SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

# Secret Managerへのアクセス権限を付与
gcloud secrets add-iam-policy-binding DATABASE_URL \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding YOUTUBE_API_KEY \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding TWITCH_CLIENT_ID \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding TWITCH_CLIENT_SECRET \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding FIREBASE_ADMIN_KEY \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/secretmanager.secretAccessor"
```

---

## 🐳 バックエンドのデプロイ (Cloud Run)

### 1. Dockerイメージのビルドとプッシュ

```bash
cd backend

# イメージをビルド・プッシュ
gcloud builds submit \
  --tag asia-northeast1-docker.pkg.dev/$GCP_PROJECT_ID/pixicast/pixicast-backend:latest
```

### 2. Cloud Runへデプロイ

```bash
gcloud run deploy pixicast-backend \
  --image asia-northeast1-docker.pkg.dev/$GCP_PROJECT_ID/pixicast/pixicast-backend:latest \
  --platform managed \
  --region asia-northeast1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 512Mi \
  --cpu 1 \
  --max-instances 10 \
  --set-secrets "\
DATABASE_URL=DATABASE_URL:latest,\
YOUTUBE_API_KEY=YOUTUBE_API_KEY:latest,\
TWITCH_CLIENT_ID=TWITCH_CLIENT_ID:latest,\
TWITCH_CLIENT_SECRET=TWITCH_CLIENT_SECRET:latest,\
GOOGLE_APPLICATION_CREDENTIALS_JSON=FIREBASE_ADMIN_KEY:latest"
```

### 3. デプロイ後の確認

```bash
# サービスURLを取得
gcloud run services describe pixicast-backend \
  --platform managed \
  --region asia-northeast1 \
  --format 'value(status.url)'

# 例: https://pixicast-backend-xxxxxxxxxx-an.a.run.app
```

### 4. CORS設定の確認

バックエンドコード (`cmd/server/main.go`) でCORS設定を確認：

```go
// 本番環境のフロントエンドドメインを追加
w.Header().Set("Access-Control-Allow-Origin", "https://your-vercel-app.vercel.app")
```

---

## ⚡ フロントエンドのデプロイ (Vercel)

### 1. Vercelプロジェクトの作成

```bash
cd frontend

# Vercel CLIでログイン
vercel login

# プロジェクトをリンク
vercel link
```

### 2. 環境変数の設定

Vercelダッシュボード → Settings → Environment Variables で以下を設定：

```bash
# Firebase設定
NEXT_PUBLIC_FIREBASE_API_KEY=AIzaSyCTVazAu9_ZHLgCFHpoPcCdJm46cBg0z3Q
NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=pixicast.firebaseapp.com
NEXT_PUBLIC_FIREBASE_PROJECT_ID=pixicast
NEXT_PUBLIC_FIREBASE_STORAGE_BUCKET=pixicast.firebasestorage.app
NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID=306954849874
NEXT_PUBLIC_FIREBASE_APP_ID=1:306954849874:web:41c828f9ad4fbc15f0198b
NEXT_PUBLIC_FIREBASE_MEASUREMENT_ID=G-3W6WCDNQNQ

# API URL (Cloud RunのURL)
NEXT_PUBLIC_API_URL=https://pixicast-backend-xxxxxxxxxx-an.a.run.app
```

### 3. デプロイ

```bash
# プロダクションデプロイ
vercel --prod
```

---

## 🔧 デプロイ後の設定

### 1. Firebase Authentication設定

Firebase Console → Authentication → Settings → **承認済みドメイン** に以下を追加：

- `your-vercel-app.vercel.app`
- `pixicast-backend-xxxxxxxxxx-an.a.run.app`

### 2. CORS設定の更新

`backend/cmd/server/main.go` のCORS設定を本番ドメインに更新：

```go
w.Header().Set("Access-Control-Allow-Origin", "https://your-vercel-app.vercel.app")
```

再デプロイ：

```bash
cd backend
gcloud builds submit --tag asia-northeast1-docker.pkg.dev/$GCP_PROJECT_ID/pixicast/pixicast-backend:latest
gcloud run deploy pixicast-backend --image asia-northeast1-docker.pkg.dev/$GCP_PROJECT_ID/pixicast/pixicast-backend:latest --region asia-northeast1
```

---

## 🧪 動作確認

### 1. バックエンドのヘルスチェック

```bash
curl https://pixicast-backend-xxxxxxxxxx-an.a.run.app/health
```

### 2. フロントエンドの動作確認

- `https://your-vercel-app.vercel.app` にアクセス
- Googleログインを試す
- チャンネル登録を試す

---

## 📊 監視とログ

### Cloud Run ログ

```bash
gcloud run services logs read pixicast-backend \
  --region asia-northeast1 \
  --limit 50
```

### Vercel ログ

Vercelダッシュボード → Deployments → 最新デプロイ → **Runtime Logs**

---

## 🔄 継続的デプロイ

### GitHub Actions (推奨)

`.github/workflows/deploy.yml` を作成して自動デプロイを設定できます。

---

## ⚠️ トラブルシューティング

### バックエンドが起動しない

```bash
# ログを確認
gcloud run services logs read pixicast-backend --region asia-northeast1 --limit 100

# Secret Managerの権限を確認
gcloud secrets get-iam-policy DATABASE_URL
```

### フロントエンドでAPIエラー

- Vercelの環境変数 `NEXT_PUBLIC_API_URL` を確認
- CORS設定を確認
- ブラウザのコンソールログを確認

### Firebase認証エラー

- Firebase Consoleで承認済みドメインを確認
- `.env.local` の設定を確認

---

## 🎉 完了！

デプロイが完了したら、以下をチェック：

- ✅ バックエンドが正常に起動
- ✅ フロントエンドが正常に表示
- ✅ Googleログインが動作
- ✅ チャンネル登録が動作
- ✅ タイムライン表示が動作

