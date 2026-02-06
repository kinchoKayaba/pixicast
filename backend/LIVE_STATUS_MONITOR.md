# ライブ配信状態監視機能

## 概要

Twitchのライブ配信が終了したときに、DBを自動的に更新するための機能です。

## 問題

- Twitchのライブ配信を開始したとき、DBに`type = 'live'`として保存される
- 配信が終了しても、DBは自動的に更新されない
- そのため、終了した配信が「放送中」と表示され続ける

## 解決策

1分ごとにTwitch APIをチェックして、配信が終了したらDBを更新する。

## 使い方

### 1. 手動実行（1回だけチェック）

```bash
cd backend
./bin/update_live_status
```

### 2. 自動監視（1分ごとにチェック）

```bash
cd backend
./bin/watch_live_status
```

このコマンドを実行すると、1分ごとにライブ配信の状態をチェックし続けます。
停止するには `Ctrl+C` を押してください。

### 3. バックグラウンドで実行

```bash
cd backend
nohup ./bin/watch_live_status > logs/live_status.log 2>&1 &
```

## 本番環境での運用

本番環境では、以下のいずれかの方法で定期実行を設定してください：

### Cloud Run (GCP) の場合

Cloud Schedulerを使用：

```bash
gcloud scheduler jobs create http live-status-checker \
  --schedule="*/1 * * * *" \
  --uri="https://your-service.run.app/internal/update-live-status" \
  --http-method=POST
```

### cronジョブの場合

```cron
# 1分ごとに実行
* * * * * cd /path/to/backend && ./bin/update_live_status >> logs/live_status.log 2>&1
```

## 処理の流れ

1. DBから現在「live」タイプのTwitchイベントを取得
2. 各イベントについて、Twitch APIで現在配信中かチェック
3. 配信が終了している場合：
   - `type`を`'video'`に変更
   - `end_at`を現在時刻に設定
   - `updated_at`を更新

## 注意事項

- Twitch API のレート制限に注意してください
- 配信中のチャンネルが多い場合、1分では処理が間に合わない可能性があります
- その場合は、実行間隔を調整するか、並列処理を検討してください

