"use client";

import { useState, useEffect } from "react";
import { useAuth } from "@/contexts/AuthContext";
import { API_BASE_URL } from "@/lib/config";

interface AddChannelModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

type Platform = "youtube" | "twitch" | "podcast";

interface PlanInfo {
  type: string;
  display_name: string;
  max_channels: number;
  current_channels: number;
}

export default function AddChannelModal({
  isOpen,
  onClose,
  onSuccess,
}: AddChannelModalProps) {
  const [platform, setPlatform] = useState<Platform>("youtube");
  const [input, setInput] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [planInfo, setPlanInfo] = useState<PlanInfo | null>(null);
  const { user, loading: authLoading, getIdToken, isAnonymous, signInWithGoogle } = useAuth();

  // プラン情報を取得
  useEffect(() => {
    if (isOpen && user) {
      fetchPlanInfo();
    }
  }, [isOpen, user]);

  const fetchPlanInfo = async () => {
    try {
      const idToken = await getIdToken();
      if (!idToken) return;

      const response = await fetch(`${API_BASE_URL}/v1/me`, {
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setPlanInfo({
          type: data.plan.type,
          display_name: data.plan.display_name,
          max_channels: data.plan.max_channels,
          current_channels: data.current_channels,
        });
      }
    } catch (error) {
      console.error("プラン情報取得エラー:", error);
    }
  };

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    // 認証チェック
    if (!user) {
      setError("ログインが必要です。しばらく待ってから再度お試しください。");
      return;
    }

    setSubmitting(true);

    try {
      const idToken = await getIdToken();
      if (!idToken) {
        throw new Error("認証トークンの取得に失敗しました。ページを更新してください。");
      }

      const response = await fetch(`${API_BASE_URL}/v1/subscriptions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${idToken}`,
        },
        body: JSON.stringify({
          platform: platform,
          input: input.trim(),
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "チャンネルの追加に失敗しました");
      }

      const data = await response.json();
      console.log("チャンネル追加成功:", data);

      // 成功時
      setInput("");
      
      // サイドバーに更新を通知
      window.dispatchEvent(new Event("channels-updated"));
      
      onSuccess();
      onClose();
    } catch (err) {
      console.error("チャンネル追加エラー:", err);
      setError(
        err instanceof Error ? err.message : "チャンネルの追加に失敗しました"
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[9999] p-4"
      onClick={handleBackdropClick}
    >
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full">
        {/* ヘッダー */}
        <div className="flex items-center justify-between p-6 border-b">
          <div>
          <h2 className="text-xl font-bold text-gray-900">
            チャンネルを追加
          </h2>
            {planInfo && (
              <div className="mt-1 flex items-center gap-2">
                <span
                  className={`text-xs px-2 py-0.5 rounded ${
                    planInfo.type === "free_anonymous"
                      ? "bg-gray-200 text-gray-700"
                      : planInfo.type === "free_login"
                      ? "bg-blue-100 text-blue-700"
                      : "bg-purple-100 text-purple-700"
                  }`}
                >
                  {planInfo.display_name}
                </span>
                <span className="text-sm text-gray-600">
                  {planInfo.current_channels} / {planInfo.max_channels}{" "}
                  チャンネル
                </span>
              </div>
            )}
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* フォーム */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {/* 認証中メッセージ */}
          {authLoading && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-3 text-sm text-blue-700">
              🔐 認証中です。しばらくお待ちください...
            </div>
          )}

          {/* プラットフォーム選択 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              プラットフォーム
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setPlatform("youtube")}
                className={`flex-1 px-4 py-2 rounded-lg border text-sm font-medium transition-colors ${
                  platform === "youtube"
                    ? "bg-red-600 text-white border-red-600"
                    : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50"
                }`}
              >
                ▶️ YouTube
              </button>
              <button
                type="button"
                onClick={() => setPlatform("twitch")}
                className={`flex-1 px-4 py-2 rounded-lg border text-sm font-medium transition-colors ${
                  platform === "twitch"
                    ? "bg-purple-600 text-white border-purple-600"
                    : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50"
                }`}
              >
                🎮 Twitch
              </button>
              <button
                type="button"
                onClick={() => setPlatform("podcast")}
                className={`flex-1 px-4 py-2 rounded-lg border text-sm font-medium transition-colors ${
                  platform === "podcast"
                    ? "bg-orange-600 text-white border-orange-600"
                    : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50"
                }`}
              >
                🎙️ Podcast
              </button>
            </div>
          </div>

          {/* 入力フィールド */}
          <div>
            <label
              htmlFor="channel-input"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              {platform === "youtube" && "YouTube URL、@ハンドル、またはチャンネルID"}
              {platform === "twitch" && "Twitch URL またはユーザー名"}
              {platform === "podcast" && "Podcast RSS URL"}
            </label>
            <input
              id="channel-input"
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={
                platform === "youtube"
                  ? "例: @junchannel または UCxxxx..."
                  : platform === "twitch"
                  ? "例: kato_junichi0817"
                  : "例: https://feeds.buzzsprout.com/xxxxx.rss"
              }
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900 bg-white placeholder:text-gray-400"
              required
              disabled={submitting || authLoading}
            />
          </div>

          {/* 例示 */}
          <div className="text-sm text-gray-600">
            <p className="font-medium mb-2">入力例:</p>
            {platform === "youtube" && (
              <ul className="space-y-1">
                <li>• https://youtube.com/@junchannel</li>
                <li>• @junchannel</li>
                <li>• UCx1nAvtVDIsaGmCMSe8ofsQ</li>
              </ul>
            )}
            {platform === "twitch" && (
              <ul className="space-y-1">
                <li>• https://www.twitch.tv/kato_junichi0817</li>
                <li>• kato_junichi0817</li>
              </ul>
            )}
            {platform === "podcast" && (
              <ul className="space-y-1">
                <li>• https://feeds.buzzsprout.com/xxxxx.rss</li>
                <li>• https://anchor.fm/s/xxxxx/podcast/rss</li>
              </ul>
            )}
          </div>

          {/* エラーメッセージ */}
          {error && (
            <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-red-700 text-sm mb-2">{error}</p>
              {/* 匿名ユーザーが上限に達した場合、ログインを促す */}
              {isAnonymous && error.includes("Freeプラン") && (
                <button
                  onClick={async () => {
                    try {
                      await signInWithGoogle();
                      onClose();
                    } catch (err) {
                      console.error("ログインエラー:", err);
                    }
                  }}
                  className="mt-2 w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
                >
                  <svg className="w-5 h-5" viewBox="0 0 24 24">
                    <path
                      fill="currentColor"
                      d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                    />
                    <path
                      fill="currentColor"
                      d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                    />
                    <path
                      fill="currentColor"
                      d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                    />
                    <path
                      fill="currentColor"
                      d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                    />
                  </svg>
                  Googleでログインしてもっと登録！
                </button>
              )}
            </div>
          )}

          {/* ボタン */}
          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors"
              disabled={submitting}
            >
              キャンセル
            </button>
            <button
              type="submit"
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:bg-gray-400 disabled:cursor-not-allowed"
              disabled={submitting || authLoading || !input.trim()}
            >
              {authLoading ? "認証中..." : submitting ? "追加中..." : "追加"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

