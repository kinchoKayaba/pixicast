#!/bin/bash

# Cloud Runデプロイスクリプト
# 使用前に以下を設定してください：
# - GCP_PROJECT_ID: Google CloudプロジェクトID
# - SERVICE_NAME: Cloud Runサービス名
# - REGION: デプロイリージョン

set -e

# 設定
PROJECT_ID="${GCP_PROJECT_ID:-pixicast}"
SERVICE_NAME="${SERVICE_NAME:-pixicast-backend}"
REGION="${REGION:-asia-northeast1}"

echo "🚀 Deploying to Cloud Run..."
echo "   Project: $PROJECT_ID"
echo "   Service: $SERVICE_NAME"
echo "   Region: $REGION"

# Artifact Registryにイメージをビルド・プッシュ
gcloud builds submit \
  --tag "asia-northeast1-docker.pkg.dev/$PROJECT_ID/pixicast/$SERVICE_NAME:latest" \
  --project="$PROJECT_ID"

# Cloud Runにデプロイ
gcloud run deploy "$SERVICE_NAME" \
  --image "asia-northeast1-docker.pkg.dev/$PROJECT_ID/pixicast/$SERVICE_NAME:latest" \
  --platform managed \
  --region "$REGION" \
  --allow-unauthenticated \
  --port 8080 \
  --memory 512Mi \
  --max-instances 10 \
  --set-secrets "DATABASE_URL=DATABASE_URL:latest,YOUTUBE_API_KEY=YOUTUBE_API_KEY:latest,TWITCH_CLIENT_ID=TWITCH_CLIENT_ID:latest,TWITCH_CLIENT_SECRET=TWITCH_CLIENT_SECRET:latest,GOOGLE_APPLICATION_CREDENTIALS_JSON=FIREBASE_ADMIN_KEY:latest" \
  --project="$PROJECT_ID"

echo "✅ Deployment complete!"
echo "🌐 Service URL:"
gcloud run services describe "$SERVICE_NAME" --platform managed --region "$REGION" --format 'value(status.url)' --project="$PROJECT_ID"

