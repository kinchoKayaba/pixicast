"use client";

import { useState, useEffect } from "react";
import { client } from "@/lib/client";
// 生成された型定義をインポート
import { Program } from "@/gen/pixicast/v1/timeline_pb";

export default function Timeline() {
  const [programs, setPrograms] = useState<Program[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // データ取得関数
    const fetchData = async () => {
      try {
        // バックエンドの GetTimeline を呼ぶ
        // YouTubeチャンネルIDも渡す（じゅんチャンネル）
        const res = await client.getTimeline({
          date: "2025-11-25",
          youtubeChannelIds: ["UCx1nAvtVDIsaGmCMSe8ofsQ"], // じゅんチャンネルのID
        });
        setPrograms(res.programs);
      } catch (error) {
        console.error("データ取得エラー:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  return (
    <main className="min-h-screen bg-pink-50 text-gray-800 p-4 pb-20">
      {/* ヘッダー */}
      <header className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-700">ホーム</h1>
        <div className="text-gray-500">🔍</div>
      </header>

      {/* 日付表示エリア */}
      <div className="flex mb-4 text-sm font-medium text-gray-500">
        <div className="mr-4 flex flex-col items-center">
          <span className="font-bold text-gray-800">11/25</span>
          <span>(月)</span>
        </div>

        <div className="border-l-2 border-gray-300 pl-4 flex-1">
          {/* ローディング表示 */}
          {loading && <p className="text-sm text-gray-400">読み込み中...</p>}

          {/* 番組リスト */}
          <div className="space-y-4">
            {programs.map((program) => (
              <div
                key={program.id}
                className="bg-white rounded-xl shadow-sm overflow-hidden flex relative h-28"
              >
                {/* 左側の色付きバー (放送中なら赤、それ以外は青) */}
                <div
                  className={`w-2 h-full absolute left-0 top-0 ${
                    program.isLive ? "bg-rose-500" : "bg-blue-500"
                  }`}
                />

                {/* 真ん中の情報エリア */}
                <div className="p-3 pl-5 flex-1 flex flex-col justify-between">
                  <div>
                    <div className="flex justify-between items-start">
                      <h2 className="text-sm font-bold text-gray-800 line-clamp-2 leading-tight">
                        {program.title}
                      </h2>
                      <button className="text-gray-300 hover:text-yellow-400">
                        ★
                      </button>
                    </div>
                    {program.isLive && (
                      <span className="inline-block bg-rose-500 text-white text-[10px] font-bold px-1.5 py-0.5 rounded mt-1">
                        放送中
                      </span>
                    )}
                  </div>

                  <div className="flex items-center text-xs text-gray-500 mt-2">
                    {/* 時間の表示 (Tで区切って時間だけ出す) */}
                    <span className="mr-2 font-mono">
                      ⏱{" "}
                      {program.startAt.includes("T")
                        ? program.startAt.split("T")[1].slice(0, 5)
                        : program.startAt}
                    </span>
                    <span className="bg-purple-100 text-purple-700 px-2 py-0.5 rounded font-bold">
                      {program.platformName}
                    </span>
                  </div>
                </div>

                {/* 右側のサムネイル画像 */}
                <div className="w-28 bg-gray-200 relative shrink-0">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={program.imageUrl || "https://placehold.jp/150x150.png"}
                    alt={program.title}
                    className="object-cover w-full h-full"
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </main>
  );
}
