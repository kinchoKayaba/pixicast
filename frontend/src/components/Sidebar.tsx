"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";

interface Channel {
  user_id: number;
  platform: string;
  source_id: string;
  channel_id: string;
  handle: string;
  display_name: string;
  thumbnail_url: string;
  enabled: boolean;
}

interface SidebarProps {
  isOpen: boolean;
  onToggle: () => void;
}

export default function Sidebar({ isOpen, onToggle }: SidebarProps) {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [collapsedPlatforms, setCollapsedPlatforms] = useState<
    Record<string, boolean>
  >({});
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const selectedChannelId = searchParams.get("channel");
  const { user, getIdToken } = useAuth();

  const togglePlatform = (platform: string) => {
    setCollapsedPlatforms((prev) => ({
      ...prev,
      [platform]: !prev[platform],
    }));
  };

  const fetchChannels = async () => {
    try {
      // 認証されていない場合は空リスト
      if (!user) {
        console.log("⚠️ Sidebar: User not authenticated");
        setChannels([]);
        return;
      }

      const idToken = await getIdToken();
      if (!idToken) {
        console.error("❌ Sidebar: Failed to get ID token");
        setChannels([]);
        return;
      }

      console.log("🔍 Sidebar: Fetching channels for user:", user.uid);
      const response = await fetch("http://localhost:8080/v1/subscriptions", {
        cache: "no-store",
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
      });

      if (!response.ok) {
        console.error("❌ Sidebar: API error:", response.status);
        setChannels([]);
        return;
      }

      const data = await response.json();
      console.log(
        "✅ Sidebar: Channels loaded:",
        data.subscriptions?.length || 0
      );
      setChannels(data.subscriptions || []);
    } catch (error) {
      console.error("❌ Sidebar: チャンネル取得エラー:", error);
      setChannels([]);
    }
  };

  // userが変わったらチャンネルを再取得
  useEffect(() => {
    fetchChannels();
  }, [pathname, user]);

  // プラットフォーム別にグループ化
  const groupedChannels = channels.reduce((acc, channel) => {
    const platform = channel.platform;
    if (!acc[platform]) {
      acc[platform] = [];
    }
    acc[platform].push(channel);
    return acc;
  }, {} as Record<string, Channel[]>);

  // プラットフォーム名を表示用に変換
  const getPlatformLabel = (platform: string) => {
    switch (platform) {
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

  // プラットフォームの色を取得
  const getPlatformColor = (platform: string) => {
    switch (platform) {
      case "youtube":
        return "text-red-600";
      case "twitch":
        return "text-purple-600";
      case "podcast":
        return "text-[#842CC2]";
      default:
        return "text-gray-600";
    }
  };

  return (
    <aside
      className={`fixed left-0 top-0 h-full bg-white border-r border-gray-200 transition-all duration-300 z-20 ${
        isOpen ? "w-64" : "w-16"
      }`}
    >
      {/* ヘッダー */}
      <div className="flex items-center p-4 border-b border-gray-200 h-16">
        {isOpen && (
          <h1 className="text-lg font-bold text-gray-800 ml-12">Pixicast</h1>
        )}
      </div>

      {/* ナビゲーション */}
      <nav className="p-2 overflow-y-auto h-[calc(100vh-64px)]">
        {/* ホーム */}
        <Link
          href="/"
          className={`flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 ${
            pathname === "/" && !selectedChannelId
              ? "bg-gray-100 font-semibold"
              : ""
          }`}
        >
          <svg
            className="w-5 h-5 flex-shrink-0"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
            />
          </svg>
          {isOpen && <span>ホーム</span>}
        </Link>

        {/* 登録チャンネル */}
        {isOpen && (
          <div className="mt-4 mb-2 px-3">
            <h3 className="text-xs font-semibold text-gray-500 uppercase">
              登録チャンネル
            </h3>
          </div>
        )}

        {/* チャンネルリスト - プラットフォーム別にグループ化 */}
        {Object.entries(groupedChannels).map(([platform, platformChannels]) => (
          <div key={platform} className="mb-3">
            {/* プラットフォームヘッダー */}
            {isOpen && (
              <button
                onClick={() => togglePlatform(platform)}
                className={`w-full flex items-center justify-between px-3 py-1 text-xs font-semibold ${getPlatformColor(
                  platform
                )} hover:opacity-80 transition-opacity`}
              >
                <span>
                  {getPlatformLabel(platform)}{" "}
                  <span className="opacity-70">
                    ({platformChannels.length})
                  </span>
                </span>
                <svg
                  className={`w-4 h-4 transition-transform ${
                    collapsedPlatforms[platform] ? "-rotate-90" : ""
                  }`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 9l-7 7-7-7"
                  />
                </svg>
              </button>
            )}

            {/* チャンネルリスト */}
            {!collapsedPlatforms[platform] &&
              platformChannels.map((channel) => (
                <Link
                  key={channel.channel_id}
                  href={`/?channel=${channel.channel_id}`}
                  className={`flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 ${
                    selectedChannelId === channel.channel_id
                      ? "bg-gray-100 font-semibold"
                      : ""
                  }`}
                >
                  <img
                    src={channel.thumbnail_url}
                    alt={channel.display_name}
                    className="w-6 h-6 rounded-full flex-shrink-0"
                  />
                  {isOpen && (
                    <span className="text-sm text-gray-700 truncate">
                      {channel.display_name}
                    </span>
                  )}
                </Link>
              ))}
          </div>
        ))}

        {/* チャンネル管理 */}
        {isOpen && <div className="border-t border-gray-200 my-2" />}
        <Link
          href="/channels"
          className={`flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 ${
            pathname === "/channels" ? "bg-gray-100 font-semibold" : ""
          }`}
        >
          <svg
            className="w-5 h-5 flex-shrink-0"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
            />
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
            />
          </svg>
          {isOpen && <span>チャンネル管理</span>}
        </Link>
      </nav>
    </aside>
  );
}
