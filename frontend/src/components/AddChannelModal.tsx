"use client";

import { useState, useEffect, useCallback } from "react";
import { useAuth } from "@/contexts/AuthContext";
import { API_BASE_URL } from "@/lib/config";
import {
  useChannelSearch,
  getRecentSearches,
  type ChannelSearchResult,
  type PopularChannelsResponse,
} from "@/hooks/useChannelSearch";
import ChannelSearchResultComponent from "@/components/ChannelSearchResult";

interface AddChannelModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

type PlatformFilter = "" | "youtube" | "twitch" | "podcast" | "radiko";

interface PlanInfo {
  type: string;
  display_name: string;
  max_channels: number;
  current_channels: number;
}

const platformFilters: { value: PlatformFilter; label: string; icon: string; activeClass: string }[] = [
  { value: "", label: "All", icon: "🔍", activeClass: "bg-gray-800 text-white border-gray-800" },
  { value: "youtube", label: "YouTube", icon: "▶️", activeClass: "bg-red-600 text-white border-red-600" },
  { value: "twitch", label: "Twitch", icon: "🎮", activeClass: "bg-purple-600 text-white border-purple-600" },
  { value: "podcast", label: "Podcast", icon: "🎙️", activeClass: "bg-orange-600 text-white border-orange-600" },
  { value: "radiko", label: "Radiko", icon: "📻", activeClass: "bg-blue-600 text-white border-blue-600" },
];

function isUrl(input: string): boolean {
  return /^https?:\/\//i.test(input.trim());
}

export default function AddChannelModal({
  isOpen,
  onClose,
  onSuccess,
}: AddChannelModalProps) {
  const { user, loading: authLoading, getIdToken, isAnonymous, signInWithGoogle } = useAuth();
  const {
    query,
    setQuery,
    platform,
    setPlatform,
    results,
    loading: searchLoading,
    error: searchError,
    quotaWarning,
    onCompositionStart,
    onCompositionEnd,
  } = useChannelSearch(getIdToken);

  const [subscribingId, setSubscribingId] = useState<string | null>(null);
  const [subscribeError, setSubscribeError] = useState("");
  const [planInfo, setPlanInfo] = useState<PlanInfo | null>(null);
  const [popularChannels, setPopularChannels] = useState<ChannelSearchResult[]>([]);
  const [urlInput, setUrlInput] = useState("");
  const [urlSubmitting, setUrlSubmitting] = useState(false);

  const isAtLimit = planInfo
    ? planInfo.current_channels >= planInfo.max_channels
    : false;

  // Fetch plan info
  useEffect(() => {
    if (!isOpen || !user) return;
    (async () => {
      try {
        const idToken = await getIdToken();
        if (!idToken) return;
        const resp = await fetch(`${API_BASE_URL}/v1/me`, {
          headers: { Authorization: `Bearer ${idToken}` },
        });
        if (resp.ok) {
          const data = await resp.json();
          setPlanInfo({
            type: data.plan.type,
            display_name: data.plan.display_name,
            max_channels: data.plan.max_channels,
            current_channels: data.current_channels,
          });
        }
      } catch (e) {
        console.error("プラン情報取得エラー:", e);
      }
    })();
  }, [isOpen, user, getIdToken]);

  // Fetch popular channels
  useEffect(() => {
    if (!isOpen || !user) return;
    (async () => {
      try {
        const idToken = await getIdToken();
        if (!idToken) return;
        const resp = await fetch(`${API_BASE_URL}/v1/channels/popular?limit=5`, {
          headers: { Authorization: `Bearer ${idToken}` },
        });
        if (resp.ok) {
          const data: PopularChannelsResponse = await resp.json();
          setPopularChannels(data.results || []);
        }
      } catch (e) {
        console.error("人気チャンネル取得エラー:", e);
      }
    })();
  }, [isOpen, user, getIdToken]);

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      setQuery("");
      setPlatform("");
      setSubscribingId(null);
      setSubscribeError("");
      setUrlInput("");
      setUrlSubmitting(false);
    }
  }, [isOpen, setQuery, setPlatform]);

  const handleSubscribe = useCallback(
    async (channel: ChannelSearchResult) => {
      if (!user) return;
      setSubscribingId(channel.external_id);
      setSubscribeError("");

      try {
        const idToken = await getIdToken();
        if (!idToken) throw new Error("認証トークンの取得に失敗しました");

        const resp = await fetch(`${API_BASE_URL}/v1/subscriptions`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${idToken}`,
          },
          body: JSON.stringify({
            platform: channel.platform_id,
            input: channel.external_id,
          }),
        });

        if (!resp.ok) {
          const data = await resp.json();
          throw new Error(data.error || "チャンネルの追加に失敗しました");
        }

        window.dispatchEvent(new Event("channels-updated"));
        onSuccess();
        onClose();
      } catch (err) {
        setSubscribeError(
          err instanceof Error ? err.message : "チャンネルの追加に失敗しました"
        );
      } finally {
        setSubscribingId(null);
      }
    },
    [user, getIdToken, onSuccess, onClose]
  );

  // Handle URL/handle direct submission
  const handleUrlSubmit = useCallback(
    async (inputValue: string) => {
      if (!user) return;
      const trimmed = inputValue.trim();
      if (!trimmed) return;

      setUrlSubmitting(true);
      setSubscribeError("");

      // Detect platform from URL
      let detectedPlatform = "youtube";
      if (/twitch\.tv/i.test(trimmed)) {
        detectedPlatform = "twitch";
      } else if (/podcasts\.apple\.com|feeds\.|\.rss|anchor\.fm/i.test(trimmed)) {
        detectedPlatform = "podcast";
      }

      try {
        const idToken = await getIdToken();
        if (!idToken) throw new Error("認証トークンの取得に失敗しました");

        const resp = await fetch(`${API_BASE_URL}/v1/subscriptions`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${idToken}`,
          },
          body: JSON.stringify({
            platform: detectedPlatform,
            input: trimmed,
          }),
        });

        if (!resp.ok) {
          const data = await resp.json();
          throw new Error(data.error || "チャンネルの追加に失敗しました");
        }

        window.dispatchEvent(new Event("channels-updated"));
        onSuccess();
        onClose();
      } catch (err) {
        setSubscribeError(
          err instanceof Error ? err.message : "チャンネルの追加に失敗しました"
        );
      } finally {
        setUrlSubmitting(false);
      }
    },
    [user, getIdToken, onSuccess, onClose]
  );

  if (!isOpen) return null;

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) onClose();
  };

  // Detect URL in search input
  const queryIsUrl = isUrl(query);
  const recentSearches = getRecentSearches();
  const showEmptyState = query.trim().length < 2 && !queryIsUrl;

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[9999] p-4"
      onClick={handleBackdropClick}
    >
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b flex-shrink-0">
          <div>
            <h2 className="text-lg font-bold text-gray-900">
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
                <span className="text-xs text-gray-600">
                  {planInfo.current_channels} / {planInfo.max_channels} チャンネル
                </span>
              </div>
            )}
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Search Input */}
        <div className="p-4 border-b flex-shrink-0">
          {authLoading && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-2 text-xs text-blue-700 mb-3">
              認証中です。しばらくお待ちください...
            </div>
          )}

          <div className="relative">
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onCompositionStart={onCompositionStart}
              onCompositionEnd={onCompositionEnd}
              placeholder="チャンネル名を検索..."
              className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900 bg-white placeholder:text-gray-400 text-sm"
              autoFocus
              disabled={authLoading}
            />
            {searchLoading && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2">
                <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
              </div>
            )}
          </div>

          {/* Platform filter pills */}
          <div className="flex gap-1.5 mt-3 overflow-x-auto">
            {platformFilters.map((pf) => (
              <button
                key={pf.value}
                type="button"
                onClick={() => setPlatform(pf.value)}
                className={`flex-shrink-0 px-3 py-1 rounded-full border text-xs font-medium transition-colors ${
                  platform === pf.value
                    ? pf.activeClass
                    : "bg-white text-gray-600 border-gray-300 hover:bg-gray-50"
                }`}
              >
                {pf.icon} {pf.label}
              </button>
            ))}
          </div>
        </div>

        {/* Results area */}
        <div className="flex-1 overflow-y-auto min-h-0">
          {/* Plan limit warning */}
          {isAtLimit && (
            <div className="mx-4 mt-3 p-3 bg-amber-50 border border-amber-200 rounded-lg">
              <p className="text-amber-700 text-xs">
                チャンネル登録数の上限に達しています（{planInfo?.current_channels}/{planInfo?.max_channels}）。
                プランをアップグレードするか、既存のチャンネルを削除してください。
              </p>
              {isAnonymous && (
                <button
                  onClick={async () => {
                    try {
                      await signInWithGoogle();
                      onClose();
                    } catch (err) {
                      console.error("ログインエラー:", err);
                    }
                  }}
                  className="mt-2 w-full flex items-center justify-center gap-2 px-3 py-1.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-xs font-medium"
                >
                  Googleでログインしてもっと登録
                </button>
              )}
            </div>
          )}

          {/* Quota warning */}
          {quotaWarning && (
            <div className="mx-4 mt-3 p-2 bg-yellow-50 border border-yellow-200 rounded-lg">
              <p className="text-yellow-700 text-xs">
                API使用量が上限に近づいています。検索結果が制限される場合があります。
              </p>
            </div>
          )}

          {/* Subscribe error */}
          {subscribeError && (
            <div className="mx-4 mt-3 p-2 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-red-700 text-xs">{subscribeError}</p>
            </div>
          )}

          {/* URL detected in search input */}
          {queryIsUrl && (
            <div className="p-4">
              <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg">
                <p className="text-sm text-blue-800 mb-2">
                  URLを検出しました。チャンネルを直接追加できます。
                </p>
                <button
                  onClick={() => handleUrlSubmit(query)}
                  disabled={urlSubmitting || isAtLimit}
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium disabled:bg-gray-400 disabled:cursor-not-allowed"
                >
                  {urlSubmitting ? "追加中..." : "このURLからチャンネルを追加"}
                </button>
              </div>
            </div>
          )}

          {/* Search error */}
          {searchError && searchError !== "url_detected" && (
            <div className="p-4">
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
                <p className="text-red-700 text-xs">{searchError}</p>
                <p className="text-red-600 text-xs mt-1">
                  URLで直接追加する場合は、チャンネルURLを入力してください。
                </p>
              </div>
            </div>
          )}

          {/* Search results */}
          {!queryIsUrl && results.length > 0 && (
            <div className="py-1">
              {results.map((channel) => (
                <ChannelSearchResultComponent
                  key={`${channel.platform_id}:${channel.external_id}`}
                  channel={channel}
                  onSubscribe={handleSubscribe}
                  subscribing={subscribingId === channel.external_id}
                  disabled={isAtLimit}
                />
              ))}
            </div>
          )}

          {/* No results */}
          {!queryIsUrl &&
            !searchLoading &&
            !searchError &&
            query.trim().length >= 2 &&
            results.length === 0 && (
              <div className="p-6 text-center">
                <p className="text-gray-500 text-sm">チャンネルが見つかりませんでした</p>
                <p className="text-gray-400 text-xs mt-1">
                  URLで直接追加することもできます
                </p>
              </div>
            )}

          {/* Empty state: Recent searches + Popular channels */}
          {showEmptyState && (
            <div className="p-4 space-y-4">
              {/* Recent searches */}
              {recentSearches.length > 0 && (
                <div>
                  <h3 className="text-xs font-medium text-gray-500 mb-2">
                    最近の検索
                  </h3>
                  <div className="space-y-1">
                    {recentSearches.map((s) => (
                      <button
                        key={s.query}
                        onClick={() => setQuery(s.query)}
                        className="w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded-lg transition-colors flex items-center gap-2"
                      >
                        <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        {s.query}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Popular channels */}
              {popularChannels.length > 0 && (
                <div>
                  <h3 className="text-xs font-medium text-gray-500 mb-2">
                    人気のチャンネル
                  </h3>
                  {popularChannels.map((channel) => (
                    <ChannelSearchResultComponent
                      key={`${channel.platform_id}:${channel.external_id}`}
                      channel={channel}
                      onSubscribe={handleSubscribe}
                      subscribing={subscribingId === channel.external_id}
                      disabled={isAtLimit}
                    />
                  ))}
                </div>
              )}

              {/* URL fallback hint */}
              <div className="text-center pt-2">
                <p className="text-xs text-gray-400">
                  URLや@ハンドルでも追加できます
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-3 border-t flex-shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors text-sm"
          >
            閉じる
          </button>
        </div>
      </div>
    </div>
  );
}
