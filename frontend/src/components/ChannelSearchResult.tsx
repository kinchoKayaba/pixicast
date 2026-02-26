"use client";

import type { ChannelSearchResult as ChannelResult } from "@/hooks/useChannelSearch";

interface ChannelSearchResultProps {
  channel: ChannelResult;
  onSubscribe: (channel: ChannelResult) => void;
  subscribing?: boolean;
  disabled?: boolean;
}

const platformConfig: Record<
  string,
  { icon: string; color: string; label: string }
> = {
  youtube: { icon: "▶️", color: "text-red-600", label: "YouTube" },
  twitch: { icon: "🎮", color: "text-purple-600", label: "Twitch" },
  podcast: { icon: "🎙️", color: "text-orange-600", label: "Podcast" },
  radiko: { icon: "📻", color: "text-blue-600", label: "Radiko" },
};

function formatCount(count: number): string {
  if (count >= 1_000_000) {
    return `${(count / 1_000_000).toFixed(1)}M`;
  }
  if (count >= 1_000) {
    return `${(count / 1_000).toFixed(1)}K`;
  }
  return count.toString();
}

export default function ChannelSearchResult({
  channel,
  onSubscribe,
  subscribing = false,
  disabled = false,
}: ChannelSearchResultProps) {
  const config = platformConfig[channel.platform_id] || {
    icon: "📺",
    color: "text-gray-600",
    label: channel.platform_id,
  };

  return (
    <div className="flex items-center gap-3 p-3 hover:bg-gray-50 rounded-lg transition-colors">
      {/* サムネイル */}
      <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gray-200 overflow-hidden">
        {channel.thumbnail_url ? (
          <img
            src={channel.thumbnail_url}
            alt={channel.display_name}
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-gray-400 text-lg">
            {config.icon}
          </div>
        )}
      </div>

      {/* チャンネル情報 */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-xs">{config.icon}</span>
          <span className="font-medium text-gray-900 truncate text-sm">
            {channel.display_name}
          </span>
        </div>
        <div className="flex items-center gap-2 text-xs text-gray-500 mt-0.5">
          {channel.handle && <span>@{channel.handle}</span>}
          {channel.subscriber_count != null && channel.subscriber_count > 0 && (
            <span>{formatCount(channel.subscriber_count)}</span>
          )}
        </div>
      </div>

      {/* 購読ボタン */}
      {channel.is_subscribed ? (
        <span className="flex-shrink-0 px-3 py-1.5 text-xs font-medium text-green-700 bg-green-50 border border-green-200 rounded-full">
          登録済み
        </span>
      ) : (
        <button
          onClick={() => onSubscribe(channel)}
          disabled={subscribing || disabled}
          className="flex-shrink-0 px-3 py-1.5 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-full transition-colors disabled:bg-gray-400 disabled:cursor-not-allowed"
        >
          {subscribing ? "追加中..." : "追加"}
        </button>
      )}
    </div>
  );
}
