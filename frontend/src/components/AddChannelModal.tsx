"use client";

import { useState } from "react";
import { useAuth } from "@/contexts/AuthContext";

interface AddChannelModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

type Platform = "youtube" | "twitch" | "podcast";

export default function AddChannelModal({
  isOpen,
  onClose,
  onSuccess,
}: AddChannelModalProps) {
  const [platform, setPlatform] = useState<Platform>("youtube");
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const { getIdToken } = useAuth();

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const idToken = await getIdToken();
      if (!idToken) {
        throw new Error("認証情報が取得できませんでした");
      }

      const response = await fetch("http://localhost:8080/v1/subscriptions", {
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
      onSuccess();
      onClose();
    } catch (err) {
      console.error("チャンネル追加エラー:", err);
      setError(
        err instanceof Error ? err.message : "チャンネルの追加に失敗しました"
      );
    } finally {
      setLoading(false);
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
          <h2 className="text-xl font-bold text-gray-900">
            チャンネルを追加
          </h2>
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
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              required
              disabled={loading}
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
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {error}
            </div>
          )}

          {/* ボタン */}
          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors"
              disabled={loading}
            >
              キャンセル
            </button>
            <button
              type="submit"
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:bg-gray-400 disabled:cursor-not-allowed"
              disabled={loading || !input.trim()}
            >
              {loading ? "追加中..." : "追加"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

