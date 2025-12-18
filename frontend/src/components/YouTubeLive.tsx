"use client";

import { useState } from "react";
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TimelineService } from "@/gen/proto/pixicast/v1/timeline_connect";
import { API_BASE_URL } from "@/lib/config";

const transport = createConnectTransport({
  baseUrl: API_BASE_URL,
});

const client = createPromiseClient(TimelineService, transport);

interface YouTubeStream {
  videoId: string;
  title: string;
  channelTitle: string;
  description: string;
  thumbnailUrl: string;
  publishedAt: string;
}

export default function YouTubeLive() {
  const [query, setQuery] = useState("ゲーム実況");
  const [streams, setStreams] = useState<YouTubeStream[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const searchStreams = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await client.searchYouTubeLive({
        query,
        maxResults: 10,
      });
      setStreams(response.streams as YouTubeStream[]);
    } catch (err) {
      setError(err instanceof Error ? err.message : "エラーが発生しました");
      console.error("Failed to search YouTube live streams:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-6xl mx-auto p-6">
      <h2 className="text-3xl font-bold mb-6">YouTubeライブ配信検索</h2>

      {/* 検索フォーム */}
      <div className="flex gap-4 mb-8">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="検索キーワードを入力"
          className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          onKeyDown={(e) => e.key === "Enter" && searchStreams()}
        />
        <button
          onClick={searchStreams}
          disabled={loading}
          className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
        >
          {loading ? "検索中..." : "検索"}
        </button>
      </div>

      {/* エラー表示 */}
      {error && (
        <div className="mb-4 p-4 bg-red-100 border border-red-400 text-red-700 rounded-lg">
          {error}
        </div>
      )}

      {/* ライブ配信一覧 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {streams.map((stream) => (
          <a
            key={stream.videoId}
            href={`https://www.youtube.com/watch?v=${stream.videoId}`}
            target="_blank"
            rel="noopener noreferrer"
            className="block bg-white rounded-lg shadow-md hover:shadow-xl transition-shadow overflow-hidden"
          >
            {/* サムネイル */}
            <div className="relative aspect-video bg-gray-200">
              {stream.thumbnailUrl && (
                <img
                  src={stream.thumbnailUrl}
                  alt={stream.title}
                  className="w-full h-full object-cover"
                />
              )}
              <div className="absolute top-2 right-2 bg-red-600 text-white px-2 py-1 text-xs font-bold rounded">
                🔴 LIVE
              </div>
            </div>

            {/* 情報 */}
            <div className="p-4">
              <h3 className="font-bold text-lg mb-2 line-clamp-2">
                {stream.title}
              </h3>
              <p className="text-gray-600 text-sm mb-2">{stream.channelTitle}</p>
              <p className="text-gray-500 text-xs line-clamp-2">
                {stream.description}
              </p>
            </div>
          </a>
        ))}
      </div>

      {/* 結果なし */}
      {!loading && streams.length === 0 && (
        <div className="text-center py-12 text-gray-500">
          検索結果がありません。キーワードを入力して検索してください。
        </div>
      )}
    </div>
  );
}

