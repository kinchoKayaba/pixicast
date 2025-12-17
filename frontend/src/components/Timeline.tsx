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
  const { user, getIdToken } = useAuth();

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

  return (
    <div className="min-h-screen bg-pink-50 text-gray-800 p-8 pt-20">
      <header className="flex justify-between items-center mb-6">
        {selectedChannelName ? (
          <div className="flex items-center gap-3">
            <div className="bg-red-600 text-white px-3 py-1.5 rounded-lg font-bold text-sm">
              YouTube
            </div>
            <h1 className="text-2xl font-bold text-gray-700">
              {selectedChannelName}
            </h1>
          </div>
        ) : (
          <h1 className="text-2xl font-bold text-gray-700">ホーム</h1>
        )}
      </header>

      <div className="space-y-8">
        {Object.entries(groupedPrograms).map(([dateKey, dayPrograms]) => {
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
                const timeStr = `${hours.toString().padStart(2, "0")}:${minutes
                  .toString()
                  .padStart(2, "0")}`;

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
                        <div className="w-2 h-full absolute left-0 top-0 bg-red-600" />

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

                        <div className="w-48 bg-gray-200 relative shrink-0 aspect-video">
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
        })}
      </div>

      {/* 無限スクロール用のトリガー要素 */}
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

      {programs.length === 0 && (
        <div className="text-center py-20">
          <p className="text-gray-500">表示する動画がありません</p>
        </div>
      )}
    </div>
  );
}
