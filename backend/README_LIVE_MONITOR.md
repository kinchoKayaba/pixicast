# ライブ配信監視機能

## 概要

Twitchのライブ配信が終了したときに、DBを自動的に更新する機能です。

## 使い方

### 開発環境での実行

#### 1. 手動で1回だけ実行

```bash
cd backend
./bin/update_live_status
```

または

```bash
cd backend
go run cmd/batch/update_live_status.go
```

#### 2. 1分ごとに自動監視（推奨）

別のターミナルウィンドウで実行：

```bash
cd backend
./bin/watch_live_status
```

このコマンドは1分ごとにライブ配信の状態をチェックし続けます。
停止するには `Ctrl+C` を押してください。

#### 3. バックグラウンドで実行

```bash
cd backend
mkdir -p logs
nohup ./bin/watch_live_status > logs/live_status.log 2>&1 &
```

バックグラウンドプロセスを確認：
```bash
ps aux | grep watch_live_status
```

停止する場合：
```bash
pkill -f watch_live_status
```

## 本番環境での設定

### Cloud Run (GCP) + Cloud Scheduler

1. バッチ用のエンドポイントを追加：

```go
// cmd/server/main.go に追加
http.HandleFunc("/internal/update-live-status", func(w http.ResponseWriter, r *http.Request) {
    // 内部リクエストのみ許可
    if r.Header.Get("X-Cloudscheduler") == "" {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // バッチ処理を実行
    // ... update_live_status.goのロジックを呼び出し
    
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})
```

2. Cloud Schedulerジョブを作成：

```bash
gcloud scheduler jobs create http live-status-updater \
  --schedule="*/1 * * * *" \
  --uri="https://your-service.run.app/internal/update-live-status" \
  --http-method=POST \
  --headers="X-Cloudscheduler=true"
```

### cronジョブ（VPS等）

```bash
# crontabを編集
crontab -e
```

以下を追加：
```cron
# 1分ごとにライブステータスを更新
* * * * * cd /path/to/pixicast/backend && ./bin/update_live_status >> logs/live_status.log 2>&1
```

## 処理の流れ

1. DBから現在`type = 'live'`のTwitchイベントを取得
2. Twitchユーザーごとにグループ化
3. 各ユーザーについて、Twitch APIで現在配信中のストリームを取得
4. DBのイベントと比較：
   - まだ配信中 → 何もしない
   - 配信終了 → DBを更新
     - `type` を `'video'` に変更
     - `end_at` を現在時刻に設定
     - `updated_at` を更新

## 出力例

```
2025/12/19 00:09:36 📺 Checking 5 live events...
2025/12/19 00:09:36 🔍 Checking Twitch user: 690460356
2025/12/19 00:09:37 ✅ Still live: アマガミ完全初見プレイやる 20日目～
2025/12/19 00:09:37 🔍 Checking Twitch user: 545050196
2025/12/19 00:09:37 🔴 Stream ended: 龍が如く極2 初見実況プレイ2日目©SEGA
2025/12/19 00:09:38 ✅ Updated: 龍が如く極2 初見実況プレイ2日目©SEGA (live -> video)
2025/12/19 00:09:38 ✅ Live status update completed. Updated 2 events.
```

## トラブルシューティング

### エラー: DATABASE_URL not set

環境変数が設定されていません。`.env.dev`ファイルを確認してください。

### エラー: TWITCH_CLIENT_ID or TWITCH_CLIENT_SECRET not set

Twitch APIの認証情報が設定されていません。`.env.dev`に以下を追加：

```bash
TWITCH_CLIENT_ID=your_client_id
TWITCH_CLIENT_SECRET=your_client_secret
```

### ライブ配信が多すぎて処理が追いつかない

実行間隔を2分または5分に延長してください：

```bash
# 60秒 → 120秒に変更
sleep 120
```

または、並列処理を実装してください。

## 注意事項

- Twitch API のレート制限: 1分あたり800リクエスト
- 配信中のチャンネルが100以上ある場合、レート制限に注意
- バッチ処理は冪等性があるため、何度実行しても安全です

