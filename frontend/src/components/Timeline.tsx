"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useSearchParams } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { client } from "@/lib/client";
// 生成された型定義をインポート
import { Program } from "@/gen/proto/pixicast/v1/timeline_pb";

export default function Timeline() {
  const [programs, setPrograms] = useState<Program[]>([]);
  const [groupedPrograms, setGroupedPrograms] = useState<
    Record<string, Program[]>
  >({});
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [currentDate, setCurrentDate] = useState(new Date());
  const searchParams = useSearchParams();
  const selectedChannelId = searchParams.get("channel");
  const [selectedChannelName, setSelectedChannelName] = useState<string>("");
  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);
  const { user, getIdToken, isAnonymous, signInWithGoogle } = useAuth();

  // フィルター状態
  const [platformFilter, setPlatformFilter] = useState<string>("all");

  // データ取得関数（初回ロード用）
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      // 認証チェック
      if (!user) {
        console.log("⚠️ Timeline: User not authenticated");
        setPrograms([]);
        setGroupedPrograms({});
        setLoading(false);
        return;
      }

      const idToken = await getIdToken();
      if (!idToken) {
        console.error("❌ Timeline: Failed to get ID token");
        setPrograms([]);
        setGroupedPrograms({});
        setLoading(false);
        return;
      }

      // 1. まず購読チャンネル一覧を取得
      const subscriptionsResponse = await fetch(
        "http://localhost:8080/v1/subscriptions",
        {
          cache: "no-store",
          headers: {
            Authorization: `Bearer ${idToken}`,
          },
        }
      );

      if (!subscriptionsResponse.ok) {
        console.error(
          "❌ Timeline: Failed to fetch subscriptions:",
          subscriptionsResponse.status
        );
        setPrograms([]);
        setGroupedPrograms({});
        setLoading(false);
        return;
      }

      const subscriptionsData = await subscriptionsResponse.json();
      let channelIds =
        subscriptionsData.subscriptions?.map(
          (sub: { channel_id: string }) => sub.channel_id
        ) || [];

      // Filter by selectedChannelId if present
      if (selectedChannelId) {
        channelIds = [selectedChannelId];
        const foundChannel = subscriptionsData.subscriptions?.find(
          (sub: { channel_id: string }) => sub.channel_id === selectedChannelId
        );
        setSelectedChannelName(foundChannel?.display_name || "");
      } else {
        setSelectedChannelName("");
      }

      console.log("📺 表示チャンネル:", channelIds);

      // 2. タイムラインを取得（初回は50件）
      const today = new Date();
      const dateStr = today.toISOString().split("T")[0];

      const res = await client.getTimeline({
        date: dateStr,
        youtubeChannelIds: channelIds,
        beforeTime: "",
        limit: 50,
      });

      // 最新順にソート
      const sortedPrograms = [...res.programs].sort((a, b) => {
        const timeA = new Date(a.startAt || a.publishedAt || 0).getTime();
        const timeB = new Date(b.startAt || b.publishedAt || 0).getTime();
        return timeB - timeA;
      });

      // 日付でグループ化
      const groupedByDate = sortedPrograms.reduce((groups, program) => {
        const date = new Date(program.startAt);
        const dateKey = `${date.getFullYear()}-${String(
          date.getMonth() + 1
        ).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
        if (!groups[dateKey]) {
          groups[dateKey] = [];
        }
        groups[dateKey].push(program);
        return groups;
      }, {} as Record<string, Program[]>);

      setPrograms(sortedPrograms);
      setGroupedPrograms(groupedByDate);
      setCurrentDate(today);
      setHasMore(res.hasMore);
      setNextCursor(res.nextCursor);
    } catch (error) {
      console.error("❌ Timeline: データ取得エラー:", error);
    } finally {
      setLoading(false);
    }
  }, [selectedChannelId, user, getIdToken]);

  // 追加データ取得関数（無限スクロール用）
  const loadMore = useCallback(async () => {
    if (loadingMore || !hasMore || !nextCursor || !user) return;

    setLoadingMore(true);
    try {
      const idToken = await getIdToken();
      if (!idToken) {
        console.error("❌ Timeline loadMore: Failed to get ID token");
        setLoadingMore(false);
        return;
      }

      // チャンネルIDを取得
      const subscriptionsResponse = await fetch(
        "http://localhost:8080/v1/subscriptions",
        {
          cache: "no-store",
          headers: {
            Authorization: `Bearer ${idToken}`,
          },
        }
      );

      if (!subscriptionsResponse.ok) {
        console.error("❌ Timeline loadMore: Failed to fetch subscriptions");
        setLoadingMore(false);
        return;
      }

      const subscriptionsData = await subscriptionsResponse.json();
      let channelIds =
        subscriptionsData.subscriptions?.map(
          (sub: { channel_id: string }) => sub.channel_id
        ) || [];

      if (selectedChannelId) {
        channelIds = [selectedChannelId];
      }

      const today = new Date();
      const dateStr = today.toISOString().split("T")[0];

      const res = await client.getTimeline({
        date: dateStr,
        youtubeChannelIds: channelIds,
        beforeTime: nextCursor,
        limit: 50,
      });

      // 既存のプログラムに追加
      const newPrograms = [...programs, ...res.programs];
      const sortedPrograms = newPrograms.sort((a, b) => {
        const timeA = new Date(a.startAt || a.publishedAt || 0).getTime();
        const timeB = new Date(b.startAt || b.publishedAt || 0).getTime();
        return timeB - timeA;
      });

      // 日付でグループ化
      const groupedByDate = sortedPrograms.reduce((groups, program) => {
        const date = new Date(program.startAt);
        const dateKey = `${date.getFullYear()}-${String(
          date.getMonth() + 1
        ).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
        if (!groups[dateKey]) {
          groups[dateKey] = [];
        }
        groups[dateKey].push(program);
        return groups;
      }, {} as Record<string, Program[]>);

      setPrograms(sortedPrograms);
      setGroupedPrograms(groupedByDate);
      setHasMore(res.hasMore);
      setNextCursor(res.nextCursor);
    } catch (error) {
      console.error("❌ Timeline: 追加データ取得エラー:", error);
    } finally {
      setLoadingMore(false);
    }
  }, [
    loadingMore,
    hasMore,
    nextCursor,
    programs,
    selectedChannelId,
    user,
    getIdToken,
  ]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Intersection Observer の設定
  useEffect(() => {
    if (loadMoreRef.current) {
      observerRef.current = new IntersectionObserver(
        (entries) => {
          if (entries[0].isIntersecting && hasMore && !loadingMore) {
            loadMore();
          }
        },
        { threshold: 0.1 }
      );

      observerRef.current.observe(loadMoreRef.current);
    }

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [hasMore, loadingMore, loadMore]);

  if (loading) {
    return (
      <div className="min-h-screen bg-pink-50 flex items-center justify-center">
        <p className="text-gray-500">読み込み中...</p>
      </div>
    );
  }

  const handleGoogleLogin = async () => {
    try {
      await signInWithGoogle();
      // ログイン後、ページをリロードしてデータを再取得
      window.location.reload();
    } catch (error) {
      console.error("Googleログインエラー:", error);
      alert("ログインに失敗しました");
    }
  };

  // プラットフォーム別の色を取得
  const getPlatformColor = (platform: string) => {
    switch (platform?.toLowerCase()) {
      case "youtube":
        return "bg-red-600";
      case "twitch":
        return "bg-purple-600";
      case "podcast":
        return "bg-[#842CC2]";
      default:
        return "bg-gray-600";
    }
  };

  // プラットフォーム名を表示用に変換
  const getPlatformLabel = (platform: string) => {
    switch (platform?.toLowerCase()) {
      case "youtube":
        return "YouTube";
      case "twitch":
        return "Twitch";
      case "podcast":
        return "Podcast";
      default:
        return platform;
    }
  };

  // 個別ページの場合、最初のプログラムからプラットフォームを取得
  const selectedPlatform =
    selectedChannelId && programs.length > 0 ? programs[0].platformName : null;

  // プログラムをフィルタリング
  const filteredPrograms = programs.filter((program) => {
    if (platformFilter === "all") return true;
    if (platformFilter === "star") return false; // TODO: お気に入り機能実装時に対応
    return program.platformName?.toLowerCase() === platformFilter;
  });

  // フィルタリング後のプログラムを日付でグループ化
  const filteredGroupedPrograms = filteredPrograms.reduce((groups, program) => {
    const date = new Date(program.startAt);
    const dateKey = `${date.getFullYear()}-${String(
      date.getMonth() + 1
    ).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
    if (!groups[dateKey]) {
      groups[dateKey] = [];
    }
    groups[dateKey].push(program);
    return groups;
  }, {} as Record<string, Program[]>);

  return (
    <div className="min-h-screen bg-pink-50 text-gray-800 p-8 pt-20">
      <header className="flex justify-between items-center mb-6">
        {selectedChannelName ? (
          <div className="flex items-center gap-3">
            <div
              className={`${
                selectedPlatform
                  ? getPlatformColor(selectedPlatform)
                  : "bg-red-600"
              } text-white px-3 py-1.5 rounded-lg font-bold text-sm`}
            >
              {selectedPlatform
                ? getPlatformLabel(selectedPlatform)
                : "YouTube"}
            </div>
            <h1 className="text-2xl font-bold text-gray-700">
              {selectedChannelName}
            </h1>
          </div>
        ) : (
          <div className="flex items-center gap-4">
            <h1 className="text-2xl font-bold text-gray-700">ホーム</h1>

            {/* フィルターボタン */}
            <div className="flex gap-2">
              <button
                onClick={() => setPlatformFilter("all")}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  platformFilter === "all"
                    ? "bg-gray-800 text-white"
                    : "bg-white text-gray-600 hover:bg-gray-100"
                }`}
              >
                すべて
              </button>
              <button
                onClick={() => setPlatformFilter("youtube")}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  platformFilter === "youtube"
                    ? "bg-red-600 text-white"
                    : "bg-white text-gray-600 hover:bg-gray-100"
                }`}
              >
                YouTube
              </button>
              <button
                onClick={() => setPlatformFilter("twitch")}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  platformFilter === "twitch"
                    ? "bg-purple-600 text-white"
                    : "bg-white text-gray-600 hover:bg-gray-100"
                }`}
              >
                Twitch
              </button>
              <button
                onClick={() => setPlatformFilter("podcast")}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  platformFilter === "podcast"
                    ? "bg-[#842CC2] text-white"
                    : "bg-white text-gray-600 hover:bg-gray-100"
                }`}
              >
                Podcast
              </button>
              <button
                onClick={() => setPlatformFilter("star")}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  platformFilter === "star"
                    ? "bg-yellow-500 text-white"
                    : "bg-white text-gray-600 hover:bg-gray-100"
                }`}
              >
                ★
              </button>
            </div>
          </div>
        )}

        {/* Firebase Authentication ログインボタン */}
        {isAnonymous && (
          <button
            onClick={handleGoogleLogin}
            className="flex items-center gap-2 bg-white border border-gray-300 px-4 py-2 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
          >
            <svg className="w-5 h-5" viewBox="0 0 24 24">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
            <span className="text-sm font-medium text-gray-700">
              Googleでログインして無制限登録！
            </span>
          </button>
        )}
      </header>

      <div className="space-y-8">
        {Object.entries(filteredGroupedPrograms).map(
          ([dateKey, dayPrograms]) => {
            const [year, month, day] = dateKey.split("-");
            const date = new Date(
              parseInt(year),
              parseInt(month) - 1,
              parseInt(day)
            );
            const dayOfWeek = ["日", "月", "火", "水", "木", "金", "土"][
              date.getDay()
            ];

            return (
              <div key={dateKey} className="space-y-4">
                <div className="flex items-center gap-4 mb-4">
                  <div className="text-center">
                    <div className="text-3xl font-bold text-gray-800">
                      {month}/{day}
                    </div>
                    <div className="text-sm text-gray-500">({dayOfWeek})</div>
                  </div>
                  <div className="flex-1 border-t border-gray-300"></div>
                </div>

                {dayPrograms.map((program) => {
                  const programDate = new Date(program.startAt);
                  const hours = programDate.getHours();
                  const minutes = programDate.getMinutes();
                  const timeStr = `${hours
                    .toString()
                    .padStart(2, "0")}:${minutes.toString().padStart(2, "0")}`;

                  return (
                    <div key={program.id} className="flex text-sm">
                      <div className="w-20 text-right pr-4 text-gray-600">
                        {timeStr}
                      </div>

                      <div className="border-l-2 border-gray-300 pl-4 flex-1">
                        <a
                          href={program.linkUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="bg-white rounded-xl shadow-sm overflow-hidden flex relative hover:shadow-md transition-shadow cursor-pointer"
                        >
                          <div
                            className={`w-2 h-full absolute left-0 top-0 ${getPlatformColor(
                              program.platformName
                            )}`}
                          />

                          <div className="p-3 pl-5 flex-1 flex flex-col justify-between">
                            <div>
                              <div className="flex justify-between items-start">
                                <h2 className="text-sm font-bold text-gray-800 line-clamp-2 leading-tight">
                                  {program.title}
                                </h2>
                                <button
                                  className="text-gray-300 hover:text-yellow-400 z-10"
                                  onClick={(e) => e.preventDefault()}
                                >
                                  ★
                                </button>
                              </div>
                              {program.isLive && (
                                <span className="inline-block bg-rose-500 text-white text-[10px] font-bold px-1.5 py-0.5 rounded mt-1">
                                  放送中
                                </span>
                              )}
                            </div>

                            <div className="flex items-center text-xs text-gray-500 mt-2 space-x-2 flex-wrap">
                              <span className="bg-purple-100 text-purple-700 px-2 py-0.5 rounded font-bold">
                                {program.platformName}
                              </span>
                              {program.channelThumbnailUrl && (
                                <img
                                  src={program.channelThumbnailUrl}
                                  alt={program.channelTitle}
                                  className="w-5 h-5 rounded-full"
                                />
                              )}
                              <span className="text-gray-700 font-medium">
                                {program.channelTitle}
                              </span>
                              {program.duration && (
                                <span className="font-mono">
                                  ⏱ {program.duration}
                                </span>
                              )}
                              {program.viewCount > 0 && (
                                <span className="font-mono">
                                  👁 {program.viewCount.toLocaleString()}
                                </span>
                              )}
                            </div>
                          </div>

                          <div
                            className={`bg-gray-200 relative shrink-0 ${
                              program.platformName?.toLowerCase() === "podcast"
                                ? "w-[108px] aspect-square"
                                : "w-48 aspect-video"
                            }`}
                          >
                            <img
                              src={
                                program.imageUrl ||
                                "https://placehold.jp/150x150.png"
                              }
                              alt={program.title}
                              className="object-cover w-full h-full"
                            />
                          </div>
                        </a>
                      </div>
                    </div>
                  );
                })}
              </div>
            );
          }
        )}
      </div>

      {/* 無限スクロール用のトリガー要素（フィルタリング後に0件の場合は非表示） */}
      {filteredPrograms.length > 0 && (
        <div
          ref={loadMoreRef}
          className="h-20 flex items-center justify-center mt-8"
        >
          {loadingMore && (
            <p className="text-gray-500 text-sm">さらに読み込み中...</p>
          )}
          {!hasMore && programs.length > 0 && (
            <p className="text-gray-400 text-sm">すべて表示しました</p>
          )}
        </div>
      )}

      {/* フィルタリング後に0件の場合のメッセージ */}
      {filteredPrograms.length === 0 && programs.length > 0 && (
        <div className="text-center py-20">
          <p className="text-gray-500">
            このフィルターでは表示する動画がありません
          </p>
        </div>
      )}

      {programs.length === 0 && (
        <div className="text-center py-20">
          <p className="text-gray-500">表示する動画がありません</p>
        </div>
      )}
    </div>
  );
}
